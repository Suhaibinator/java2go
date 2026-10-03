package transpiler

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func staticImportInventoryLegalContext(t *testing.T, ordinaryImports int) Ctx {
	t.Helper()
	sources := map[string]string{
		"q/Owner.java": `package q;public class Owner {public static int first(){return 1;}public static int Box=3;public static class Box{}static class Hidden{}}`,
		"q/Other.java": `package q;public class Other {public static int second(){return 2;}}`,
		"q/Peer.java":  `package q;import static q.Owner.*;public class Peer{}`,
	}
	var source strings.Builder
	source.WriteString(`package p;import static q.Owner.first;import static q.Owner.*;import static q.Owner.first;import static q.Other.second;import static q.Owner.Box;`)
	for index := 0; index < ordinaryImports; index++ {
		name := fmt.Sprintf("Noise%03d", index)
		fmt.Fprintf(&source, "import q.%s;", name)
		sources["q/"+name+".java"] = "package q;public class " + name + "{}"
	}
	source.WriteString(`public class Active<T>{class Box{}}`)
	sources["p/Active.java"] = source.String()
	ctx := sourceGenericViewDemandTestContext(t, sources)
	active := findQualifiedSourceClass("p.Active")
	ctx.currentFile = findFileScopeForClassScope(active)
	ctx.currentClass = active
	ctx.callableSubclasses = &callableSubclassSourceInventory{}
	if active == nil || len(allSourceClassScopes()) != ordinaryImports+7 {
		t.Fatal("complete legal source declarations were not retained")
	}
	return ctx
}

func staticImportInventoryExpectedTuples() []staticMethodImport {
	return []staticMethodImport{{owner: "q.Owner", member: "first"}, {owner: "q.Owner", wildcard: true}, {owner: "q.Other", member: "second"}, {owner: "q.Owner", member: "Box"}}
}

func TestStaticImportInventory_QueriesReuseOnlySourceTuples(t *testing.T) {
	t.Run("ordered-duplicate-and-wildcard-tuples", func(t *testing.T) {
		ctx := staticImportInventoryLegalContext(t, 64)
		for query := 0; query < 3; query++ {
			if !reflect.DeepEqual(staticMethodImports(ctx), staticImportInventoryExpectedTuples()) {
				t.Fatal("ordered static import tuples or duplicate elimination changed")
			}
		}
	})
	t.Run("declaring-file-and-source-pairing", func(t *testing.T) {
		ctx := staticImportInventoryLegalContext(t, 0)
		_ = staticMethodImports(ctx)
		owner := findQualifiedSourceClass("q.Owner")
		changedClass := ctx.Clone()
		changedClass.currentClass = owner
		if !reflect.DeepEqual(staticMethodImports(changedClass), staticImportInventoryExpectedTuples()) {
			t.Fatal("invocation class displaced the declaring file's import tree")
		}
		ownerCtx := classScopeCtx(owner, ctx)
		if ownerCtx.currentFile == ctx.currentFile || len(staticMethodImports(ownerCtx)) != 0 {
			t.Fatal("hierarchy context paired one file's tree with another file's bytes")
		}
		peerCtx := classScopeCtx(findQualifiedSourceClass("q.Peer"), ctx)
		if !reflect.DeepEqual(staticMethodImports(peerCtx), []staticMethodImport{{owner: "q.Owner", wildcard: true}}) {
			t.Fatal("another source file inherited cached caller imports")
		}
	})
	t.Run("fresh-namespace-access-and-lexical-shadow", func(t *testing.T) {
		ctx := staticImportInventoryLegalContext(t, 0)
		_ = staticMethodImports(ctx)
		owner := findQualifiedSourceClass("q.Owner")
		box, ok := staticImportedSourceType("Box", false, ctx)
		if !ok || box == nil || box.Enclosing != owner {
			t.Fatal("field and member-type namespaces were conflated")
		}
		if got, ok := staticImportedSourceType("first", false, ctx); ok || got != nil {
			t.Fatal("static method import invented a member type")
		}
		if got, ok := staticImportedSourceType("Hidden", true, ctx); ok || got != nil {
			t.Fatal("package-private member type became accessible across packages")
		}
		peerCtx := classScopeCtx(findQualifiedSourceClass("q.Peer"), ctx)
		if got, ok := staticImportedSourceType("Hidden", true, peerCtx); !ok || got == nil || got.Enclosing != owner {
			t.Fatal("fresh same-package accessibility was not evaluated")
		}
		lexical := resolveClassScopeByQualifiedName(ctx, "Box")
		if lexical == nil || lexical == box || lexical.Enclosing != ctx.currentClass || ctx.currentClass.TypeParameters[0].Declaration == nil {
			t.Fatal("import tuples displaced a lexical declaration or owner binder")
		}
	})
	t.Run("fresh-graph-with-reused-inventory", func(t *testing.T) {
		ctx := staticImportInventoryLegalContext(t, 0)
		oldOwner := findQualifiedSourceClass("q.Owner")
		_ = staticMethodImports(ctx)
		fresh := staticImportInventoryLegalContext(t, 0)
		fresh.callableSubclasses = ctx.callableSubclasses
		owner := findQualifiedSourceClass("q.Owner")
		got, ok := staticImportedSourceType("Box", false, fresh)
		if owner == oldOwner || !ok || got == nil || got.Enclosing != owner || got.Enclosing == oldOwner || !reflect.DeepEqual(staticMethodImports(fresh), staticImportInventoryExpectedTuples()) {
			t.Fatal("fresh graph retained old import owner or source identities")
		}
	})
	t.Run("repeated-query-budget-includes-first-inventory", func(t *testing.T) {
		ctx := staticImportInventoryLegalContext(t, 512)
		var result []staticMethodImport
		measure := func(queries int) float64 {
			return testing.AllocsPerRun(3, func() {
				// Every measured batch starts cold, including its first complete
				// import inventory. Setup and the full resolved graph stay intact.
				ctx.callableSubclasses = &callableSubclassSourceInventory{}
				for query := 0; query < queries; query++ {
					result = staticMethodImports(ctx)
				}
			})
		}
		one, repeated := measure(1), measure(32)
		if !reflect.DeepEqual(result, staticImportInventoryExpectedTuples()) || symbol.GlobalScope != ctx.callableSubclasses.graph && ctx.callableSubclasses.graph != nil {
			t.Fatal("measured queries changed import tuples or source graph")
		}
		t.Logf("same complete519class graph/512ordinary+4static imports: one cold query allocations=%.0f cold32query batch allocations=%.0f ratio=%.2f", one, repeated, repeated/one)
		if one == 0 || repeated > 6*one {
			t.Fatalf("STATIC_IMPORT_TUPLES_REPEATED_AST_SCAN: %.0f allocations >6*same graph one cold query %.0f", repeated, one)
		}
	})
}
