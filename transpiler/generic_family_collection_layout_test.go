package transpiler

import (
	"os"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func collectionLayoutTestOccurrence(t *testing.T, source, owner, written string, ordinal int, header bool) (Ctx, *sitter.Node) {
	t.Helper()
	helper := setupParseHelper(t, source)
	scope := helper.File.Symbols.FindClassScope(owner)
	if scope == nil {
		t.Fatal("missing exact source owner")
	}
	ctx := classScopeCtx(scope, helper.Ctx)
	var matches []*sitter.Node
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if (node.Type() == "generic_type" || node.Type() == "array_type" || node.Type() == "type_identifier") && node.Content([]byte(source)) == written {
			matches = append(matches, node)
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			visit(child)
		}
	}
	visit(scope.Class.DeclarationNode)
	if ordinal < 0 {
		ordinal = len(matches) - 1
	}
	if ordinal < 0 || ordinal >= len(matches) {
		t.Fatalf("missing occurrence %d of %q", ordinal, written)
	}
	occurrence := matches[ordinal]
	if header {
		ctx = classHeaderTypeCtx(scope, ctx)
	} else {
		for _, method := range scope.Methods {
			if method.DeclarationNode != nil && receiverNodeWithin(occurrence, method.DeclarationNode) {
				ctx.localScope = method
				break
			}
		}
	}
	return ctx, occurrence
}

func TestGenericFamilyCollectionArgumentIndependentOwnership(t *testing.T) {
	for _, test := range []struct {
		name, source, written string
		ordinal               int
		header, want          bool
	}{
		{"import", `import java.util.Collection; class Owner<E>{Collection<E> value;}`, "Collection<E>", 0, false, true},
		{"qualified", `class Owner<E>{java.util.Collection<E> value;}`, "java.util.Collection<E>", 0, false, true},
		{"raw", `import java.util.Collection; class Owner<E>{Collection value;}`, "Collection", 0, false, true},
		{"source", `class Collection<X>{} class Owner<E>{Collection<E> value;}`, "Collection<E>", 0, false, false},
		{"class_binder", `class Owner<Collection>{Collection value;}`, "Collection", -1, false, false},
		{"method_binder", `class Owner<E>{<Collection> void use(){Collection value=null;}}`, "Collection", -1, false, false},
		{"unknown_import", `import example.Collection; class Owner<E>{Collection<E> value;}`, "Collection<E>", 0, false, false},
		{"ordered_local_before", `import java.util.Collection; class Owner<E>{void use(){Collection<E> before=null;class Collection<X>{}Collection<E> after=null;}}`, "Collection<E>", 0, false, true},
		{"ordered_local_after", `import java.util.Collection; class Owner<E>{void use(){Collection<E> before=null;class Collection<X>{}Collection<E> after=null;}}`, "Collection<E>", 1, false, false},
		{"member", `import java.util.Collection; class Owner<E>{class Collection<X>{}Collection<E> value;}`, "Collection<E>", 0, false, false},
		{"header_own_member", `import java.util.Collection; class Base<A>{}class Owner<E> extends Base<Collection<E>>{class Collection<X>{}}`, "Collection<E>", 0, true, true},
		{"inherited_member", `class Base{static class Collection<X>{}}class Owner<E> extends Base{Collection<E> value;}`, "Collection<E>", 0, false, false},
		{"qualified_source_prefix", `class java{static class util{static class Collection<X>{}}}class Owner<E>{java.util.Collection<E> value;}`, "java.util.Collection<E>", 0, false, false},
		{"array", `import java.util.Collection; class Owner<E>{Collection<E>[] value;}`, "Collection<E>[]", 0, false, false},
		{"arity", `import java.util.Collection; class Owner<E>{Collection<E,E> value;}`, "Collection<E,E>", 0, false, false},
		{"list", `class Owner<E>{java.util.List<E> value;}`, "java.util.List<E>", 0, false, false},
		{"array_list", `class Owner<E>{java.util.ArrayList<E> value;}`, "java.util.ArrayList<E>", 0, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, node := collectionLayoutTestOccurrence(t, test.source, "Owner", test.written, test.ordinal, test.header)
			if got := genericFamilyCollectionArgumentIndependent(test.written, ctx, node); got != test.want {
				t.Fatalf("native declaration/layout proof=%v want %v", got, test.want)
			}
			if genericFamilyCollectionArgumentIndependent(test.written, ctx) || genericFamilyCollectionArgumentIndependent(test.written, ctx, node, node) {
				t.Fatal("absent/ambiguous occurrence donated a native layout")
			}
		})
	}
}

func TestGenericFamilyCollectionCarrierCompletePlan(t *testing.T) {
	data, err := os.ReadFile("testdata/generic_collection_carrier/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	helper := setupParseHelper(t, string(data))
	carrier := helper.File.Symbols.FindClassScope("Carrier")
	many := helper.File.Symbols.FindClassScope("Many")
	if carrier == nil || many == nil {
		t.Fatal("missing declaration graph")
	}
	plan, err := planGenericFamily(carrier, helper.Ctx)
	if err != nil || plan == nil {
		t.Fatalf("existing Collection protocol lacks complete family proof: %v", err)
	}
	if _, member := plan.members[many]; !member {
		t.Fatal("specialized child was omitted")
	}
	if many.Superclass != "Carrier<Collection<E>>" {
		t.Fatalf("logical specialization changed: %s", many.Superclass)
	}
	ctx, node := collectionLayoutTestOccurrence(t, string(data), "Many", "Collection<E>", 0, true)
	if !genericFamilyCollectionArgumentIndependent("Collection<E>", ctx, node) {
		t.Fatal("native Collection declaration not proven")
	}
	for _, argument := range []string{"E", "String", "java.lang.Number"} {
		if got := symbol.NodeToStr(collectionTypeExpr("Collection", []string{argument}, inScopeTypeParameters(ctx), ctx)); !strings.HasSuffix(got, ".JavaIterable") {
			t.Fatalf("argument %s changed native storage: %s", argument, got)
		}
	}
}

func TestGenericFamilyCollectionProofKeepsUnsupportedLayoutsClosed(t *testing.T) {
	for _, typ := range []string{"java.util.List<E>", "java.util.ArrayList<E>", "java.util.Collection<E>[]"} {
		t.Run(typ, func(t *testing.T) {
			helper := setupParseHelper(t, "class Carrier<A>{A value;}class Many<E> extends Carrier<"+typ+">{}")
			carrier := helper.File.Symbols.FindClassScope("Carrier")
			if plan, err := planGenericFamily(carrier, helper.Ctx); plan != nil || err == nil {
				t.Fatal("unproved native storage admitted with its family")
			}
		})
	}
}

func TestGenericFamilyCollectionOccurrenceCannotBorrowForeignRoot(t *testing.T) {
	const source = `import java.util.Collection; class Owner<E>{Collection<E> value;}`
	ctx, node := collectionLayoutTestOccurrence(t, source, "Owner", "Collection<E>", 0, false)
	if !genericFamilyCollectionArgumentIndependent("Collection<E>", ctx, node) {
		t.Fatal("own parsed occurrence lost its native protocol")
	}
	_, foreign := collectionLayoutTestOccurrence(t, source, "Owner", "Collection<E>", 0, false)
	if genericFamilyCollectionArgumentIndependent("Collection<E>", ctx, foreign) {
		t.Fatal("identical text from a foreign parse tree donated a layout")
	}
	synthetic := ctx
	synthetic.currentFile = &symbol.FileScope{Source: ctx.currentFile.Source, Imports: ctx.currentFile.Imports, BaseClass: ctx.currentFile.BaseClass, TopLevelClasses: ctx.currentFile.TopLevelClasses}
	if genericFamilyCollectionArgumentIndependent("Collection<E>", synthetic, node) {
		t.Fatal("mutable class aliases substituted for the original parse anchor")
	}
}

func TestGenericFamilyCollectionClassProofCannotDonateUnknownStorage(t *testing.T) {
	const source = `class Owner<T>{java.lang.Class<foreign.Collection<T>> hidden;}`
	ctx, outer := collectionLayoutTestOccurrence(t, source, "Owner", "java.lang.Class<foreign.Collection<T>>", 0, false)
	if !canonicalClassArgumentIndependent("java.lang.Class<foreign.Collection<T>>", ctx, outer) {
		t.Fatal("outer Class lost its own metadata-only layout proof")
	}
	var collection *sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "generic_type" && node.Content(ctx.currentFile.Source) == "foreign.Collection<T>" {
			collection = node
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(outer)
	if collection == nil || genericFamilyCollectionArgumentIndependent("foreign.Collection<T>", ctx, collection) {
		t.Fatal("unknown Collection borrowed the outer Class proof")
	}
	if plan, err := planGenericFamily(ctx.currentClass, ctx); plan != nil || err == nil {
		t.Fatal("Class layout alone admitted unknown owned nested storage")
	}
}

func TestGenericFamilyCollectionDescriptorBridgeOwnsCompleteDeclaration(t *testing.T) {
	data, err := os.ReadFile("testdata/generic_collection_carrier/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	helper := setupParseHelper(t, string(data))
	carrier := helper.File.Symbols.FindClassScope("Carrier")
	many := helper.File.Symbols.FindClassScope("Many")
	if carrier == nil || many == nil {
		t.Fatal("missing actual causal declarations")
	}
	for _, method := range carrier.Methods {
		if method.OriginalName != "read" && method.OriginalName != "write" {
			continue
		}
		family, known := genericFamilyMethodPlan(carrier, method, helper.Ctx)
		if !known || family == nil {
			t.Fatalf("%s lost complete erased dispatch family", method.OriginalName)
		}
		var bridge *directOwnerOverrideBridgePlan
		for i := range family.overrides {
			if family.overrides[i].owner == many {
				bridge = &family.overrides[i]
			}
		}
		if bridge == nil {
			t.Fatalf("%s lacks actual Collection specialization bridge", method.OriginalName)
		}
		definition := bridge.method
		if method.OriginalName == "write" {
			if len(bridge.parameters) != 1 || !bridge.parameters[0].requiresCast || !genericFamilyIsObjectErasure(carrier, bridge.parameters[0].erasedJavaType, helper.Ctx) || boundResultBridgeGoType(t, javaTypeStringToGoTypeExpr(bridge.parameters[0].erasedJavaType, nil, classScopeCtx(carrier, helper.Ctx))) != "any" {
				t.Fatalf("Collection write bridge lost nominal cast-before-mutation: %+v", bridge.parameters)
			}
			definition = bridge.method.Parameters[0]
		} else if !bridge.result.requiresWidening || !genericFamilyIsObjectErasure(carrier, bridge.result.erasedJavaType, helper.Ctx) || boundResultBridgeGoType(t, javaTypeStringToGoTypeExpr(bridge.result.erasedJavaType, nil, classScopeCtx(carrier, helper.Ctx))) != "any" {
			t.Fatalf("Collection read bridge lost Object dispatch result: %+v", bridge.result)
		}
		if got, known := genericFamilyCollectionDefinitionErasure(many, definition, helper.Ctx); !known || got != "java.util.Collection" {
			t.Fatalf("%s declaration lost exact JVM erasure: %s", method.OriginalName, got)
		}
		copy := *definition
		if _, known := genericFamilyCollectionDefinitionErasure(many, &copy, helper.Ctx); known {
			t.Fatal("copied definition borrowed another declaration's bridge")
		}
	}
}

func TestGenericFamilyCollectionDescriptorCannotAdmitPartialFamily(t *testing.T) {
	helper := setupParseHelper(t, `import java.util.Collection;abstract class Carrier<A>{abstract A read();}class Many<E> extends Carrier<Collection<E>>{java.util.List<E> unsupported;Collection<E> read(){return null;}}`)
	many := helper.File.Symbols.FindClassScope("Many")
	if many == nil {
		t.Fatal("missing partial family owner")
	}
	for _, method := range many.Methods {
		if method.OriginalName == "read" {
			if _, known := genericFamilyCollectionDefinitionErasure(many, method, helper.Ctx); known {
				t.Fatal("Collection descriptor forced admission of unsupported sibling storage")
			}
		}
	}
}
