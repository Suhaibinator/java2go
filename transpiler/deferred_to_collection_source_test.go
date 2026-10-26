package transpiler

import (
	"go/ast"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestDeferredToCollectionFactorySourceActualBody(t *testing.T) {
	for _, args := range []string{"", "first,null,last"} {
		file := sourceArrayListP2GuardFile(t, `import java.util.stream.Stream;import java.util.stream.Collectors;`+sourceArrayListNativeGuardRoot+`class Use{Values<String> run(String first,String last){return Stream.<String>of(`+args+`).collect(Collectors.toCollection(Values<String>::new));}}`)
		body := sourceArrayListP2GuardText(sourceArrayListP2GuardMethod(t, file, "RunJava2goExecution").Body)
		if strings.Count(body, "StreamFromValues[*stdjava.JavaString](") != 1 || strings.Contains(body, "NewStream[") || strings.Contains(body, "NewExecution(") || !strings.Contains(body, "StreamCollectPlanExecution(__java2goExecution") {
			t.Fatalf("actual public body collector source not initialized: %s", body)
		}
	}
}

func TestDeferredToCollectionFactorySourceOperandOnceAndOrder(t *testing.T) {
	effectHelper, effectCtx, effectNode := sourceArrayListA4CollectorInvocation(t, `class Effects{static String firstEffect(){return null;}static String lastEffect(){return null;}}`, "ArrayList<String>", `Stream.<String>of(Effects.firstEffect(),null,Effects.lastEffect()).collect(Collectors.toCollection(ArrayList<String>::new))`)
	actual := ParseExpr(effectNode, effectHelper.File.Source, effectCtx)
	actualReceiver := effectNode.ChildByFieldName("object")
	for _, index := range []int{0, 2} {
		expected := ParseExpr(invocationArgumentNode(actualReceiver, index), effectHelper.File.Source, effectCtx)
		count := 0
		ast.Inspect(actual, func(n ast.Node) bool {
			if reflect.DeepEqual(n, expected) {
				count++
			}
			return true
		})
		if count != 1 {
			t.Fatalf("actual Java effect operand %d rendered %d times", index, count)
		}
	}
	h, ctx, node := sourceArrayListA4CollectorInvocation(t, "", "ArrayList<String>", `Stream.<String>of(first,null,last).collect(Collectors.toCollection(ArrayList<String>::new))`)
	recv := ParseExpr(node.ChildByFieldName("object"), h.File.Source, ctx).(*ast.CallExpr)
	effects := []ast.Expr{&ast.CallExpr{Fun: ast.NewIdent("firstEffect")}, &ast.CallExpr{Fun: ast.NewIdent("nullEffect")}, &ast.CallExpr{Fun: ast.NewIdent("lastEffect")}}
	recv.Args = effects
	before := sourceArrayListP2GuardText(recv)
	got, ok := deferredToCollectionFactorySource(recv, node, ctx, h.File.Source)
	if !ok {
		t.Fatal("actual zero-frame factory declined")
	}
	call := got.(*ast.CallExpr)
	if call == recv || !reflect.DeepEqual(call.Fun, &ast.IndexExpr{X: stdjavaQualifiedExpr("StreamFromValues", ctx), Index: recv.Fun.(*ast.IndexExpr).Index}) || sourceArrayListP2GuardText(recv) != before {
		t.Fatal("factory shell changed original or wrong qualification")
	}
	for i, item := range effects {
		if call.Args[i] != item {
			t.Fatal("argument reparsed, copied or reordered")
		}
	}
	rendered := sourceArrayListP2GuardText(got)
	if strings.Index(rendered, "firstEffect()") > strings.Index(rendered, "nullEffect()") || strings.Index(rendered, "nullEffect()") > strings.Index(rendered, "lastEffect()") {
		t.Fatal("argument order changed")
	}
	for _, name := range []string{"firstEffect()", "nullEffect()", "lastEffect()"} {
		if strings.Count(rendered, name) != 1 {
			t.Fatal("argument duplicated")
		}
	}
}

func TestDeferredToCollectionFactorySourcePhysicalGuards(t *testing.T) {
	h, ctx, node := sourceArrayListA4CollectorInvocation(t, "", "ArrayList<String>", `Stream.<String>of(first,null,last).collect(Collectors.toCollection(ArrayList<String>::new))`)
	fresh := func() *ast.CallExpr {
		return ParseExpr(node.ChildByFieldName("object"), h.File.Source, ctx).(*ast.CallExpr)
	}
	for _, mutate := range []func(*ast.CallExpr){
		func(c *ast.CallExpr) { c.Ellipsis = token.Pos(1) },
		func(c *ast.CallExpr) { c.Args = c.Args[:1] },
		func(c *ast.CallExpr) { c.Fun.(*ast.IndexExpr).Index = ast.NewIdent("any") },
		func(c *ast.CallExpr) {
			c.Fun.(*ast.IndexExpr).X = &ast.SelectorExpr{X: ast.NewIdent("foreign"), Sel: ast.NewIdent("NewStream")}
		},
	} {
		call := fresh()
		mutate(call)
		if _, ok := deferredToCollectionFactorySource(call, node, ctx, h.File.Source); ok {
			t.Fatal("unowned physical factory admitted")
		}
	}
	for _, expr := range []string{`Stream.<String>of(first).collect(Collectors.toCollection(ArrayList<String>::new))`, `Stream.<String>of(first,null,last).collect(Collectors.toList())`} {
		other, c, n := sourceArrayListA4CollectorInvocation(t, "", "Object", expr)
		if _, ok := deferredToCollectionFactorySource(ParseExpr(n.ChildByFieldName("object"), other.File.Source, c), n, c, other.File.Source); ok {
			t.Fatal("single argument or other collector adapted")
		}
	}
}

func TestDeferredToCollectionFactorySourceAuthorityGuards(t *testing.T) {
	h, ctx, node := sourceArrayListA4CollectorInvocation(t, "", "ArrayList<String>", `Stream.<String>of(first,null,last).collect(Collectors.toCollection(ArrayList<String>::new))`)
	recv := ParseExpr(node.ChildByFieldName("object"), h.File.Source, ctx)
	registered := symbol.GlobalScope
	_, foreign, foreignNode := sourceArrayListA4CollectorInvocation(t, "", "ArrayList<String>", `Stream.<String>of(first,null,last).collect(Collectors.toCollection(ArrayList<String>::new))`)
	foreignReceiver := ParseExpr(foreignNode.ChildByFieldName("object"), foreign.currentFile.Source, foreign)
	if _, ok := deferredToCollectionFactorySource(foreignReceiver, foreignNode, foreign, foreign.currentFile.Source); !ok {
		t.Fatal("foreign tree did not establish its own positive premise")
	}
	symbol.GlobalScope = registered
	if _, ok := deferredToCollectionFactorySource(recv, node, ctx, h.File.Source); !ok {
		t.Fatal("original attached graph not restored before authority negatives")
	}
	for _, row := range []struct {
		ctx    Ctx
		source []byte
	}{{foreign, h.File.Source}, {ctx, append([]byte(nil), h.File.Source...)}, {ctx, nil}, {ctx, h.File.Source[:1]}} {
		if _, ok := deferredToCollectionFactorySource(recv, node, row.ctx, row.source); ok {
			t.Fatal("foreign executable or source bytes admitted")
		}
	}
	if _, ok := deferredToCollectionFactorySource(recv, foreignNode, ctx, h.File.Source); ok {
		t.Fatal("foreign occurrence borrowed source")
	}
	for _, decl := range []string{`class Stream{}`, `class Collectors{}`, `class String{}`} {
		other, c, n := sourceArrayListA4CollectorInvocation(t, decl, "Object", `Stream.<String>of(first,null,last).collect(Collectors.toCollection(ArrayList<java.lang.String>::new))`)
		if _, ok := deferredToCollectionFactorySource(recv, n, c, other.File.Source); ok {
			t.Fatal("source namespace donor admitted")
		}
	}
}

func TestDeferredToCollectionFactorySourceRetainsWrittenObjectAndBinders(t *testing.T) {
	h, ctx, node := sourceArrayListA4CollectorInvocation(t, `class Plain extends java.util.ArrayList<Object>{}`, "Plain", `Stream.<Object>of(first,null,last).collect(Collectors.toCollection(Plain::new))`)
	got, ok := deferredToCollectionFactorySource(ParseExpr(node.ChildByFieldName("object"), h.File.Source, ctx), node, ctx, h.File.Source)
	if !ok || !strings.Contains(sourceArrayListP2GuardText(got), "StreamFromValues[any](") {
		t.Fatal("written Object replaced by operand String LUB")
	}
	source := sourceArrayListNativeGuardRoot + `class Parent<A,B>{}class Use<X,Y> extends Parent<Y,X>{Values<X> run(X first,X last){return java.util.stream.Stream.<X>of(first,null,last).collect(java.util.stream.Collectors.toCollection(Values<X>::new));}}`
	other := setupParseHelper(t, source)
	owner := other.Ctx.currentFile.FindClassScope("Use")
	method := owner.FindMethodByName("run", nil)
	c := classScopeCtx(owner, other.Ctx)
	c.localScope = method
	n := findNode(method.DeclarationNode.ChildByFieldName("body"), "method_invocation")
	if _, ok := deferredToCollectionFactorySource(ParseExpr(n.ChildByFieldName("object"), other.File.Source, c), n, c, other.File.Source); !ok {
		t.Fatal("actual reordered owner binder lost")
	}
}
