package transpiler

import (
	"reflect"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func nativeCollectionDiamondActual(t *testing.T, source string) (Ctx, *sitter.Node) {
	t.Helper()
	h := setupParseHelper(t, source)
	owner := h.File.Symbols.FindClassScope("Use")
	if owner == nil {
		t.Fatal("missing original owner")
	}
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = owner.FindMethodByName("make", nil)
	node := findNode(h.File.Ast, "object_creation_expression")
	if node == nil {
		t.Fatal("missing original creation")
	}
	ctx.expectedTypeRoot = node
	declarator := node.Parent()
	for declarator != nil && declarator.Type() == "parenthesized_expression" {
		declarator = declarator.Parent()
	}
	if declarator != nil && declarator.Type() == "variable_declarator" && declarator.Parent() != nil {
		if typ := declarator.Parent().ChildByFieldName("type"); typ != nil {
			ctx.expectedType = typ.Content(h.File.Source)
		}
	}
	return ctx, node
}

func TestNativeCollectionDiamondCanonicalTarget(t *testing.T) {
	for _, source := range []string{
		`class Use{static void make(){java.util.Collection<String> value=new java.util.ArrayList<>();}}`,
		`import java.util.Collection;import java.util.ArrayList;class Use{static void make(){Collection<String> value=new ArrayList<>();}}`,
		`class Use<T>{void make(){java.util.Collection<T> value=new java.util.ArrayList<>();}}`,
		`class Use<T extends Number>{void make(){java.util.Collection<T> value=new java.util.ArrayList<>();}}`,
		`class Use{<T extends Number>void make(){java.util.Collection<T> value=new java.util.ArrayList<>();}}`,
		`class Bound{} class Use<T extends Bound>{void make(){java.util.Collection<T> value=new java.util.ArrayList<>();}}`,
		`class Item{} class Use{void make(){java.util.Collection<Item> value=new java.util.ArrayList<>();}}`,
		`class Use{void make(){java.util.Collection<String[]> value=new java.util.ArrayList<>();}}`,
		`class Use{void make(){java.util.Collection<int[]> value=new java.util.ArrayList<>();}}`,
		`class Use{void make(){class String{} java.util.Collection<java.lang.String> value=new java.util.ArrayList<>();}}`,
		`class Use{void make(){java.util.Collection<String> value=new java.util.ArrayList<>();class String{}}}`,
	} {
		ctx, node := nativeCollectionDiamondActual(t, source)
		typ := node.ChildByFieldName("type")
		base := typ.NamedChild(0).Content(ctx.currentFile.Source)
		args, ok := nativeCollectionDiamondTargetArguments(ctx.expectedType, base, ctx, node)
		_, want := parseJavaTypeString(ctx.expectedType)
		if !ok || !reflect.DeepEqual(args, want) {
			t.Fatalf("actual canonical parent target refused: %v/%v want %v", args, ok, want)
		}
		args[0] = "mutated return"
		again, ok := nativeCollectionDiamondTargetArguments(ctx.expectedType, base, ctx, node)
		if !ok || !reflect.DeepEqual(again, want) {
			t.Fatal("query retained mutable returned arguments")
		}
	}
}

func TestNativeCollectionDiamondShapeAndNamespaceRefusals(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"rawCreation", `class Use{void make(){java.util.Collection<String> value=new java.util.ArrayList();}}`},
		{"rawTarget", `class Use{void make(){java.util.Collection value=new java.util.ArrayList<>();}}`},
		{"varTarget", `class Use{void make(){var value=new java.util.ArrayList<>();}}`},
		{"objectTarget", `class Use{void make(){Object value=new java.util.ArrayList<>();}}`},
		{"noTarget", `class Use{void make(){new java.util.ArrayList<>();}}`},
		{"wildcard", `class Use{void make(){java.util.Collection<?> value=new java.util.ArrayList<>();}}`},
		{"upperWildcard", `class Use{void make(){java.util.Collection<? extends Number> value=new java.util.ArrayList<>();}}`},
		{"nested", `class Use{void make(){java.util.Collection<java.util.List<String>> value=new java.util.ArrayList<>();}}`},
		{"wrongArity", `class Use{void make(){java.util.Collection<String,Integer> value=new java.util.ArrayList<>();}}`},
		{"primitiveArgument", `class Use{void make(){java.util.Collection<int> value=new java.util.ArrayList<>();}}`},
		{"voidArgument", `class Use{void make(){java.util.Collection<void> value=new java.util.ArrayList<>();}}`},
		{"unknownArgument", `class Use{void make(){java.util.Collection<Missing> value=new java.util.ArrayList<>();}}`},
		{"localArgument", `class Use{void make(){class String{} java.util.Collection<String> value=new java.util.ArrayList<>();}}`},
		{"memberArgument", `class Use{class String{} void make(){java.util.Collection<String> value=new java.util.ArrayList<>();}}`},
		{"localArgumentPrefix", `class Use{void make(){class java{} java.util.Collection<java.lang.String> value=new java.util.ArrayList<>();}}`},
		{"unrelatedSet", `class Use{void make(){java.util.Set<String> value=new java.util.ArrayList<>();}}`},
		{"foreignImport", `import vendor.Collection;class Use{void make(){Collection<String> value=new java.util.ArrayList<>();}}`},
		{"sourceTarget", `class Collection<T>{} class Use{void make(){Collection<String> value=new java.util.ArrayList<>();}}`},
		{"sourceConstructor", `class ArrayList<T>{} class Use{void make(){java.util.Collection<String> value=new ArrayList<>();}}`},
		{"localTarget", `class Use{void make(){class Collection<T>{} Collection<String> value=new java.util.ArrayList<>();}}`},
		{"localConstructor", `class Use{void make(){class ArrayList<T>{} java.util.Collection<String> value=new ArrayList<>();}}`},
		{"methodBinder", `class Use{<Collection>void make(){Collection<String> value=new java.util.ArrayList<>();}}`},
		{"qualifiedPrefix", `class java{static class util{static class Collection<T>{}}} class Use{void make(){java.util.Collection<String> value=new java.util.ArrayList<>();}}`},
		{"explicitArgument", `class Use{void make(){java.util.Collection<String> value=new java.util.ArrayList<String>();}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, node := nativeCollectionDiamondActual(t, tc.source)
			typ := node.ChildByFieldName("type")
			base, _ := parseJavaTypeString(typ.Content(ctx.currentFile.Source))
			if _, ok := nativeCollectionDiamondTargetArguments(ctx.expectedType, base, ctx, node); ok {
				t.Fatal("unproved target or namespace acquired native parent inference")
			}
		})
	}
}

func TestNativeCollectionDiamondLifetimeRefusals(t *testing.T) {
	const source = `class Use{void make(){java.util.Collection<String> value=new java.util.ArrayList<>();}}`
	for _, mode := range []string{"nilRoot", "nilFile", "nilOwner", "nilCallable", "wrongType", "wrongConstructed", "foreignRoot"} {
		t.Run(mode, func(t *testing.T) {
			ctx, node := nativeCollectionDiamondActual(t, source)
			constructed := "java.util.ArrayList"
			switch mode {
			case "nilRoot":
				ctx.expectedTypeRoot = nil
			case "nilFile":
				ctx.currentFile = nil
			case "nilOwner":
				ctx.currentClass = nil
			case "nilCallable":
				ctx.localScope = nil
			case "wrongType":
				ctx.expectedType = "java.util.Collection<Integer>"
			case "wrongConstructed":
				constructed = "ArrayList"
			case "foreignRoot":
				other, otherNode := nativeCollectionDiamondActual(t, source)
				_ = other
				ctx.expectedTypeRoot = otherNode
			}
			if _, ok := nativeCollectionDiamondTargetArguments(ctx.expectedType, constructed, ctx, node); ok {
				t.Fatal("stale caller context acquired original target authority")
			}
		})
	}
}

func TestNativeCollectionDiamondExistingInferenceControls(t *testing.T) {
	for _, tc := range []struct{ source, want string }{
		{`class Use{static void make(){java.util.List<String> values=new java.util.ArrayList<>();}}`, "stdjava.NewArrayList[*stdjava.JavaString]()"},
		{`class Use{static void make(){java.util.Collection<String> values=new java.util.ArrayList<String>();}}`, "stdjava.NewArrayList[*stdjava.JavaString]()"},
		{`class Use{static void make(){java.util.Map<String,Integer> values=new java.util.HashMap<>();}}`, "stdjava.NewMap[*stdjava.JavaString, *stdjava.Integer]()"},
	} {
		out := renderGoFileFromJava(t, tc.source)
		if !strings.Contains(out, tc.want) {
			t.Fatalf("existing inference changed; want %s in:\n%s", tc.want, out)
		}
	}
}

func TestNativeCollectionDiamondSyntheticArgumentRefusal(t *testing.T) {
	ctx, node := nativeCollectionDiamondActual(t, `class Use{void make(){java.util.Collection<Missing> value=new java.util.ArrayList<>();}}`)
	ctx.syntheticTypeParameters = []symbol.TypeParam{symbol.NewTypeParam("Missing", nil)}
	if _, ok := nativeCollectionDiamondTargetArguments(ctx.expectedType, "java.util.ArrayList", ctx, node); ok {
		t.Fatal("unproved caller synthetic binder became an original Java argument")
	}
	if len(ctx.syntheticTypeParameters) != 1 {
		t.Fatal("pure query changed caller bindings")
	}
}

func TestNativeCollectionDiamondCallerPrefixRefusals(t *testing.T) {
	for _, source := range []string{
		`class Use{void make(){java.util.Collection<String> value=new java.util.ArrayList<>();class String{}}}`,
		`class Use{void make(){{class String{}}java.util.Collection<String> value=new java.util.ArrayList<>();}}`,
	} {
		ctx, node := nativeCollectionDiamondActual(t, source)
		later := findNode(ctx.localScope.DeclarationNode, "class_declaration")
		if later == nil {
			t.Fatal("missing retained local declaration")
		}
		ctx.localClasses = map[string]*localClassInfo{"String": {scope: &symbol.ClassScope{Class: &symbol.Definition{OriginalName: "String", DeclarationNode: later}}}}
		if _, ok := nativeCollectionDiamondTargetArguments(ctx.expectedType, "java.util.ArrayList", ctx, node); ok {
			t.Fatal("stale later/sibling local argument binding reached physical conversion")
		}
	}
	ctx, node := nativeCollectionDiamondActual(t, `class Use<T>{void make(){java.util.Collection<T> value=new java.util.ArrayList<>();}}`)
	ctx.syntheticTypeParameters = []symbol.TypeParam{symbol.NewTypeParam("T", nil)}
	if _, ok := nativeCollectionDiamondTargetArguments(ctx.expectedType, "java.util.ArrayList", ctx, node); ok {
		t.Fatal("synthetic prefix conflicting with original T reached physical conversion")
	}
}
