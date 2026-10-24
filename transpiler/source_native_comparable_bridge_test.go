package transpiler

import (
	"go/ast"
	"strconv"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestNativeComparableBridgeDeclarationOwnerAndAliases(t *testing.T) {
	h := setupParseHelper(t, `package p; class Item{} class Value<T> implements java.lang.Comparable<Item>{public int compareTo(Item x){return 7;} public int compareTo(String x){return 8;}}`)
	scope := h.File.Symbols.FindClassScope("Value")
	ctx := h.Ctx.Clone()
	ctx.currentClass = scope
	ctx.className = scope.Class.Name
	scope.TypeParameters[0].Declaration.GoName = "Item"
	ctx.syntheticTypeParameters = []symbol.TypeParam{symbol.NewTypeParam("java", nil)}
	ctx.localScope = &symbol.Definition{TypeParameters: ctx.syntheticTypeParameters}
	names := make(map[*symbol.Definition]string)
	for _, method := range scope.Methods {
		names[method] = method.Name
	}
	plan := planSourceNativeComparableBridge(ctx)
	if plan == nil || plan.owner != scope || plan.method.Parameters[0].OriginalType != "Item" || plan.transport != nil {
		t.Fatal("source Item was captured by emitted alias/caller binder or overload")
	}
	pointer, ok := plan.castType.(*ast.StarExpr)
	if !ok {
		t.Fatal("missing concrete declared erasure pointer")
	}
	name, ok := pointer.X.(*ast.Ident)
	if !ok || name.Name != h.File.Symbols.FindClassScope("Item").Class.Name {
		t.Fatal("wrong nominal erased type")
	}
	for method, name := range names {
		if method.Name != name {
			t.Fatal("planner renamed a source method")
		}
	}
}

func TestNativeComparableBridgeDeclinesUnprovedContractOwners(t *testing.T) {
	cases := []string{
		`package p;interface Comparable<T>{int compareTo(T x);}class Item{}class Value implements Comparable<Item>{public int compareTo(Item x){return 1;}}`,
		`package p;class java{static class lang{interface Comparable<T>{int compareTo(T x);}}}class Item{}class Value implements java.lang.Comparable<Item>{public int compareTo(Item x){return 1;}}`,
		`package p;import unknown.*;class Item{}class Value implements java.lang.Comparable<Item>{public int compareTo(Item x){return 1;}}`,
		`package p;class Item{}class Value<java> implements java.lang.Comparable<Item>{public int compareTo(Item x){return 1;}}`,
	}
	for _, source := range cases {
		h := setupParseHelper(t, source)
		scope := h.File.Symbols.FindClassScope("Value")
		ctx := h.Ctx.Clone()
		ctx.currentClass = scope
		ctx.className = scope.Class.Name
		if plan := planSourceNativeComparableBridge(ctx); plan != nil {
			t.Fatalf("unproved owner acquired native entry: %s", source)
		}
	}
}

func TestNativeComparableBridgeDeclaredErasureAndStringOnlyRegistration(t *testing.T) {
	h := setupParseHelper(t, `package p;class Item{}class Other extends Item{}class Value<T extends Item> implements java.lang.Comparable<T>{public int compareTo(T x){return 7;}}`)
	scope := h.File.Symbols.FindClassScope("Value")
	ctx := h.Ctx.Clone()
	ctx.currentClass = scope
	ctx.className = scope.Class.Name
	plan := planSourceNativeComparableBridge(ctx)
	if plan == nil {
		t.Fatal("missing declared bound bridge")
	}
	pointer, ok := plan.castType.(*ast.StarExpr)
	if !ok {
		t.Fatal("erasure is not a concrete bound pointer")
	}
	name, ok := pointer.X.(*ast.Ident)
	if !ok || name.Name != h.File.Symbols.FindClassScope("Item").Class.Name {
		t.Fatal("cast used Go T or Other instead of declared Item")
	}
	// On this prefix the unchanged implementation has T constrained exactly to
	// *Item. The bridge transports the checked Item; completed physical family
	// erasure can remove this Go-only conversion without changing the Java cast.
	if _, physical := directOwnerMethodParameterInterfaceErasure(scope, plan.method, 0, classScopeCtx(scope, ctx)); !physical && canonicalGenericFamily(scope, ctx) == nil {
		if target, ok := plan.transport.(*ast.Ident); !ok || target.Name != scope.TypeParameters[0].EmittedName() {
			t.Fatal("missing exact constrained physical transport")
		}
	}
	declarations := generateSourceNativeComparableBridgeDecls(ctx)
	if len(declarations) != 2 {
		t.Fatal("missing bridge/metadata pair")
	}
	bridge, ok := declarations[0].(*ast.FuncDecl)
	if !ok {
		t.Fatal("missing bridge function")
	}
	registration, ok := declarations[1].(*ast.GenDecl)
	if !ok {
		t.Fatal("missing registration declaration")
	}
	calls := 0
	ast.Inspect(registration, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "RegisterSourceErasedContract" {
			return true
		}
		calls++
		if len(call.Args) != 4 {
			t.Fatal("registry payload arity")
		}
		owner, ok := call.Args[0].(*ast.CallExpr)
		if !ok || len(owner.Args) != 1 {
			t.Fatal("owner must be literal TypeID")
		}
		if _, ok := owner.Args[0].(*ast.BasicLit); !ok {
			t.Fatal("generic owner instantiated at package init")
		}
		descriptor, ok := call.Args[2].(*ast.BasicLit)
		if !ok || descriptor.Value != strconv.Quote("compareTo(Ljava/lang/Object;)I") {
			t.Fatal("wrong erased descriptor")
		}
		entry, ok := call.Args[3].(*ast.BasicLit)
		if !ok || entry.Value != strconv.Quote(bridge.Name.Name) {
			t.Fatal("registration did not name exact allocated entry")
		}
		return false
	})
	if calls != 1 {
		t.Fatal("registry calls must be exactly one")
	}
}
