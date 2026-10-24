package transpiler

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestCollectionResultDemand183RawElementsOnce(t *testing.T) {
	if collectionDemand183Isolated(t, 1) {
		return
	}
	source := `import java.util.Map; class Probe {Map raw; void run(){raw.entrySet();}}`
	h := setupParseHelper(t, source)
	defer runtime.KeepAlive(h)
	ctx := h.Ctx.Clone()
	ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
	node := collectionDemand183Invocation(t, h.File.Ast, []byte(source), "raw.entrySet()")
	owner, known := intrinsicReceiverTypeName(node.ChildByFieldName("object"), ctx, []byte(source))
	if !known || owner != "JavaMap" {
		t.Fatal("raw Map owner proof lost")
	}
	elements := receiverElementJavaTypes(node.ChildByFieldName("object"), ctx, []byte(source))
	if len(elements) != 0 {
		t.Fatal("raw control must actually have nil/empty elements")
	}
	name, _ := collectionDemand183Native(t, func() string { return node.ChildByFieldName("name").Content([]byte(source)) })
	one, _ := collectionDemand183Native(t, func() string {
		return strings.Join(receiverElementJavaTypes(node.ChildByFieldName("object"), ctx, []byte(source)), ",")
	})
	native, result := collectionDemand183Native(t, func() string {
		v, k := intrinsicCollectionMethodResultType(node, owner, ctx, []byte(source))
		return fmt.Sprintf("%s/%t", v, k)
	})
	if result != "java.util.Set<java.util.Map.Entry<Object, Object>>/true" {
		t.Fatalf("raw fallback changed: %s", result)
	}
	encoded, _ := json.Marshal(collectionDemand183Work{"raw.entrySet()", owner, native, name, one, result, true})
	t.Log("COLLECTION_DEMAND_WORK183 " + string(encoded))
	if one == 0 || native > name+one {
		t.Fatalf("nil/empty receiver elements must derive at most once: native%d name%d one%d", native, name, one)
	}
}

func TestCollectionResultDemand183PublicOwnerAliasState(t *testing.T) {
	files := physicalInterface181Graph(t, map[string]string{
		"left/dup/Item.java":  `package left.dup; public class Item {public int value;}`,
		"right/dup/Item.java": `package right.dup; public class Item {public long value;}`,
		"donor/Holder.java":   `package donor; public class Holder<T> { public java.util.List<T> items; }`,
		"caller/Main.java": `package caller;
class dup {}
class List {public long size(){return 7L;}}
public class Main {void run(donor.Holder<left.dup.Item> left, donor.Holder<right.dup.Item> right,List source){left.items.size();left.items.get(0);right.items.size();right.items.get(0);source.size();}
 <T extends List> long bound(T source){return source.size();}
}`,
	})
	file := files["caller/Main.java"]
	scope := file.Symbols.FindClassScope("Main")
	if scope == nil {
		t.Fatal("actual caller declaration missing")
	}
	ctx := Ctx{currentFile: file.Symbols, currentClass: scope, localScope: scope.FindMethodByName("run", nil), executionContextName: "execution", importAliases: map[string]string{}, usedImports: map[string]bool{}}
	for _, call := range []string{"left.items.size()", "left.items.get(0)", "right.items.size()", "right.items.get(0)"} {
		node := collectionDemand183Invocation(t, file.Ast, file.Source, call)
		owner, known := intrinsicMethodReceiverTypeName(node.ChildByFieldName("object"), node.ChildByFieldName("name").Content(file.Source), ctx, file.Source)
		if !known || owner != "List" {
			t.Fatalf("declaration-owned native List was rebound in caller: %s/%t", owner, known)
		}
		aliasesBefore := map[string]string{}
		for k, v := range ctx.importAliases {
			aliasesBefore[k] = v
		}
		usedBefore := map[string]bool{}
		for k, v := range ctx.usedImports {
			usedBefore[k] = v
		}
		result, ok := inferIntrinsicMethodResultType(node, ctx, file.Source)
		if !ok {
			t.Fatalf("actual public intrinsic query lost %s", call)
		}
		if !reflect.DeepEqual(aliasesBefore, ctx.importAliases) || !reflect.DeepEqual(usedBefore, ctx.usedImports) {
			t.Fatalf("redundant query changed caller import reservation: before%v/%v after%v/%v", aliasesBefore, usedBefore, ctx.importAliases, ctx.usedImports)
		}
		if strings.HasSuffix(call, "size()") {
			if result != "int" {
				t.Fatalf("canonical size result %s", result)
			}
		} else {
			want := "left.dup.Item"
			if strings.HasPrefix(call, "right") {
				want = "right.dup.Item"
			}
			if result != want {
				t.Fatalf("donor type argument identity: %s want%s", result, want)
			}
			typ := javaTypeStringToGoTypeExpr(result, inScopeTypeParameters(ctx), ctx)
			selected := false
			ast.Inspect(typ, func(n ast.Node) bool {
				if s, yes := n.(*ast.SelectorExpr); yes && s.Sel.Name == "Item" {
					if id, yes := s.X.(*ast.Ident); yes && id.Name == ctx.importAliases[strings.TrimSuffix(want, ".Item")] {
						selected = true
					}
				}
				return true
			})
			if !selected {
				t.Fatalf("rendered result lost the actual owned Item selector: %s", physicalInterface181String(t, typ))
			}
		}
		expression := ParseExpr(node, file.Source, ctx)
		t.Logf("PUBLIC_DEMAND183 %s result=%s expr=%s aliases=%v used=%v", call, result, physicalInterface181String(t, expression), ctx.importAliases, ctx.usedImports)
	}
	left, right := ctx.importAliases["left.dup"], ctx.importAliases["right.dup"]
	if left == "" || right == "" || left == right || left == "dup" || right == "dup" || !ctx.usedImports["left.dup"] || !ctx.usedImports["right.dup"] {
		t.Fatalf("actual alias collision/required packages not preserved: %v/%v", ctx.importAliases, ctx.usedImports)
	}
	for _, method := range []string{"run", "bound"} {
		local := ctx.Clone()
		local.localScope = scope.FindMethodByName(method, nil)
		node := collectionDemand183Invocation(t, local.localScope.DeclarationNode, file.Source, "source.size()")
		if owner, known := intrinsicMethodReceiverTypeName(node.ChildByFieldName("object"), "size", local, file.Source); known {
			t.Fatalf("source/binder owner became intrinsic: %s", owner)
		}
		if result, known := inferExprJavaType(node, local, file.Source); !known || result != "long" {
			t.Fatalf("source/binder declaration result changed: %s/%t", result, known)
		}
	}
}

func TestCollectionResultDemand183PublicOptionalTargets(t *testing.T) {
	source := `import java.util.Optional; class Probe {Optional<String> value;void run(){value.map(x->null);}}`
	h := setupParseHelper(t, source)
	defer runtime.KeepAlive(h)
	base := h.Ctx.Clone()
	base.localScope = base.currentClass.FindMethodByName("run", nil)
	node := collectionDemand183Invocation(t, h.File.Ast, []byte(source), "value.map(x->null)")
	for _, tc := range []struct {
		target, want string
		match        bool
	}{{"Optional<String>", "Optional<String>", true}, {"Optional<Integer>", "Optional<Integer>", true}, {"Optional<String>", "Optional<Object>", false}} {
		ctx := base.Clone()
		ctx.expectedType = tc.target
		ctx.expectedTypeRoot = h.File.Ast
		if tc.match {
			ctx.expectedTypeRoot = node
		}
		result, known := inferIntrinsicMethodResultType(node, ctx, []byte(source))
		if !known || result != tc.want {
			t.Fatalf("public per-call target %s/match%t: %s/%t want%s", tc.target, tc.match, result, known, tc.want)
		}
	}
}
