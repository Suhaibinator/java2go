package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"strings"
	"testing"
)

const sourceArrayListNativeEqualityP1 = `class Root extends java.util.ArrayList<java.lang.Object>{} class Child extends Root{} `

func sourceArrayListNativeEqualityRender(t *testing.T, declarations, expression string) string {
	t.Helper()
	h := setupParseHelper(t, declarations+`class Use{boolean run(Child owner,Root base,java.util.ArrayList<java.lang.Object> array,java.util.List<java.lang.Object> list){return `+expression+`;}}`)
	owner := h.Ctx.currentFile.FindClassScope("Use")
	method := owner.FindMethodByName("run", nil)
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	statement := findNode(method.DeclarationNode, "return_statement")
	node := nodeutil.SemanticNamedChild(statement, 0)
	return sourceArrayListP2GuardText(ParseExpr(node, h.File.Source, ctx))
}
func TestSourceArrayListP2NativeSourceEqualityActualRendering(t *testing.T) {
	for _, expression := range []string{"array==owner", "owner==array", "list==owner", "owner==list", "array!=owner", "owner!=array", "list!=owner", "owner!=list", "array==base", "base!=list"} {
		t.Run(expression, func(t *testing.T) {
			got := sourceArrayListNativeEqualityRender(t, sourceArrayListNativeEqualityP1, expression)
			if !strings.Contains(got, "JavaReferenceEqual(") {
				t.Fatalf("actual Java native/source equality retained Go pointer comparison: %s", got)
			}
			if strings.Contains(expression, "!=") && !strings.Contains(got, "!stdjava.JavaReferenceEqual") {
				t.Fatalf("Java != lost negated identity predicate: %s", got)
			}
		})
	}
}
func TestSourceArrayListP2NativeSourceEqualityNullAndHierarchyControls(t *testing.T) {
	for _, expression := range []string{"owner==null", "null==array", "list!=null", "null!=owner", "owner==base", "base!=owner"} {
		got := sourceArrayListNativeEqualityRender(t, sourceArrayListNativeEqualityP1, expression)
		if !strings.Contains(got, "JavaReferenceEqual(") {
			t.Fatalf("existing null/hierarchy reference equality regressed: %s", got)
		}
	}
}
func TestSourceArrayListP2NativeSourceEqualityAllocationViews(t *testing.T) {
	h := setupParseHelper(t, sourceArrayListNativeEqualityP1+`class Use{boolean run(){Child owner=new Child();java.util.ArrayList<java.lang.Object> same=owner;java.util.List<java.lang.Object> distinct=new Child();return (same==owner)&&(owner!=distinct)&&(new Child()!=same);}}`)
	owner := h.Ctx.currentFile.FindClassScope("Use")
	method := owner.FindMethodByName("run", nil)
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	root := nodeutil.SemanticNamedChild(findNode(method.DeclarationNode, "return_statement"), 0)
	value := ParseExpr(root, h.File.Source, ctx)
	rendered := sourceArrayListP2GuardText(value)
	if strings.Count(rendered, "JavaReferenceEqual(") != 3 {
		t.Fatalf("same and distinct allocation views require three identity predicates: %s", rendered)
	}
	// Pure ParseExpr has not materialized the class constructor API. Derive
	// this actual operand's emitted AST through that same existing API rather
	// than hard-coding the name used by full-program declaration emission.
	effect := findNode(root, "object_creation_expression")
	if effect == nil {
		t.Fatal("actual effectful Java allocation operand absent")
	}
	effectText := sourceArrayListP2GuardText(ParseExpr(effect, h.File.Source, ctx))
	occurrences := 0
	ast.Inspect(value, func(node ast.Node) bool {
		if expr, ok := node.(ast.Expr); ok && sourceArrayListP2GuardText(expr) == effectText {
			occurrences++
		}
		return true
	})
	if occurrences != 1 {
		t.Fatalf("effectful comparison operand must occur once, actual operand=%s count=%d: %s", effectText, occurrences, rendered)
	}
}
func TestSourceArrayListP2NativeEqualitySourceHomonymControl(t *testing.T) {
	h := setupParseHelper(t, `class ArrayList<T>{} class List<T>{} class Use{boolean run(ArrayList<java.lang.Object> first,ArrayList<java.lang.Object> second){return first==second;}}`)
	owner := h.Ctx.currentFile.FindClassScope("Use")
	method := owner.FindMethodByName("run", nil)
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	node := findNode(method.DeclarationNode, "binary_expression")
	got := sourceArrayListP2GuardText(ParseExpr(node, h.File.Source, ctx))
	if strings.Contains(got, "JavaReferenceEqual(") {
		t.Fatalf("same-named source type acquired canonical native alias identity authority: %s", got)
	}
}

func TestSourceArrayListP2NativeEqualityInheritedElementOrigins(t *testing.T) {
	for _, source := range []string{
		sourceArrayListNativeGuardRoot + `class Parent{static class Item{} Values<Item> owner;java.util.List<Item> nativeAlias;} class Child extends Parent{static class Item{} boolean run(){return owner==nativeAlias;}}`,
		sourceArrayListNativeGuardRoot + `class Parent<A,B>{Values<B> owner;java.util.List<B> nativeAlias;} class Child<X,Y> extends Parent<Y,X>{boolean run(){return owner==nativeAlias;}}`,
	} {
		h := setupParseHelper(t, source)
		owner := h.Ctx.currentFile.FindClassScope("Child")
		method := owner.FindMethodByName("run", nil)
		ctx := classScopeCtx(owner, h.Ctx)
		ctx.localScope = method
		node := findNode(method.DeclarationNode, "binary_expression")
		got := sourceArrayListP2GuardText(ParseExpr(node, h.File.Source, ctx))
		if !strings.Contains(got, "JavaReferenceEqual(") {
			t.Fatalf("inherited field origin recaptured as caller homonym/binder: %s", got)
		}
	}
}
func TestSourceArrayListP2NativeEqualityDeclarationAuthorityGuards(t *testing.T) {
	for _, row := range []struct{ name, parameters, prefix string }{
		{"invariant_mismatch", `Values<java.lang.String> owner,java.util.List<java.lang.Object> nativeAlias`, ""},
		{"source_suffix", `Values<java.lang.String> owner[],java.util.List<java.lang.String> nativeAlias`, ""},
		{"native_suffix", `Values<java.lang.String> owner,java.util.List<java.lang.String> nativeAlias[]`, ""},
		{"source_homonym", `Values<java.lang.String> owner,List<java.lang.String> nativeAlias`, `class List<T>{}`},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, sourceArrayListNativeGuardRoot+row.prefix+`class Use{boolean run(`+row.parameters+`){return owner==nativeAlias;}}`)
			owner := h.Ctx.currentFile.FindClassScope("Use")
			method := owner.FindMethodByName("run", nil)
			ctx := classScopeCtx(owner, h.Ctx)
			ctx.localScope = method
			node := findNode(method.DeclarationNode, "binary_expression")
			if sourceArrayListNativeEquality(node.ChildByFieldName("left"), node.ChildByFieldName("right"), ctx, h.File.Source) {
				t.Fatal("unowned or invariant-mismatched origin acquired native identity authority")
			}
		})
	}
	source := sourceArrayListNativeGuardRoot + `class Use{boolean a(Values<java.lang.String> owner,java.util.List<java.lang.String> nativeAlias){return owner==nativeAlias;}boolean b(Values<java.lang.String> owner,java.util.List<java.lang.String> nativeAlias){return false;}}`
	h := setupParseHelper(t, source)
	owner := h.Ctx.currentFile.FindClassScope("Use")
	method := owner.FindMethodByName("a", nil)
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	node := findNode(method.DeclarationNode, "binary_expression")
	left, right := node.ChildByFieldName("left"), node.ChildByFieldName("right")
	if !sourceArrayListNativeEquality(left, right, ctx, h.File.Source) {
		t.Fatal("actual compatible scalar declarations lost identity authority")
	}
	wrong := ctx.Clone()
	wrong.localScope = owner.FindMethodByName("b", nil)
	if sourceArrayListNativeEquality(left, right, wrong, h.File.Source) {
		t.Fatal("wrong executable borrowed identity authority")
	}
	if sourceArrayListNativeEquality(left, right, ctx, append([]byte(nil), h.File.Source...)) {
		t.Fatal("copied caller source acquired actual occurrence authority")
	}
	graph := symbol.GlobalScope
	foreign := setupParseHelper(t, source)
	foreignMethod := foreign.Ctx.currentFile.FindClassScope("Use").FindMethodByName("a", nil)
	symbol.GlobalScope = graph
	original := method.DeclarationNode
	method.DeclarationNode = foreignMethod.DeclarationNode
	if sourceArrayListNativeEquality(left, right, ctx, h.File.Source) {
		t.Fatal("foreign rebound executable acquired identity authority")
	}
	method.DeclarationNode = original
}
