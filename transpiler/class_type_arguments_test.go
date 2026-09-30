package transpiler

import (
	"reflect"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestNormalizeClassTypeArguments_CarriesEnclosingReceiverBeforeDeclaredArguments(t *testing.T) {
	root := symbol.TypeParam{Name: "T"}
	value := symbol.TypeParam{Name: "V", Bounds: []symbol.JavaType{{Original: "Root"}}}
	outer := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{root},
		DeclaredTypeParameters: []symbol.TypeParam{root},
	}
	child := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{root, value},
		DeclaredTypeParameters: []symbol.TypeParam{value},
	}

	tests := []struct {
		name         string
		source       []string
		receiverArgs []string
		want         []string
	}{
		{name: "symbolic receiver", source: []string{"Impl"}, want: []string{"T", "Impl"}},
		{name: "concrete receiver", source: []string{"Impl"}, receiverArgs: []string{"String"}, want: []string{"String", "Impl"}},
		{name: "fully qualified", source: []string{"String", "Impl"}, receiverArgs: []string{"Ignored"}, want: []string{"String", "Impl"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := normalizeClassTypeArguments(child, test.source, outer, test.receiverArgs)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("normalized arguments = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestNormalizeClassTypeArguments_RawDeclaredArgumentUsesTransitiveFirstBoundErasure(t *testing.T) {
	root := symbol.TypeParam{Name: "B", Bounds: []symbol.JavaType{{Original: "Root"}}}
	value := symbol.TypeParam{Name: "U", Bounds: []symbol.JavaType{{Original: "B"}}}
	scope := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{root, value},
		DeclaredTypeParameters: []symbol.TypeParam{root, value},
	}

	got := normalizeClassTypeArguments(scope, nil, nil, nil)
	want := []string{"Root", "Root"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("raw arguments = %#v, want %#v", got, want)
	}
}

func TestNormalizeClassTypeArguments_NonGenericInnerCarriesOnlyOuterArgument(t *testing.T) {
	outerParameter := symbol.TypeParam{Name: "T"}
	outer := &symbol.ClassScope{TypeParameters: []symbol.TypeParam{outerParameter}}
	leaf := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{outerParameter},
		DeclaredTypeParameters: []symbol.TypeParam{},
	}

	got := normalizeClassTypeArguments(leaf, nil, outer, []string{"String"})
	want := []string{"String"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("carried arguments = %#v, want %#v", got, want)
	}
}

func TestNormalizeClassTypeArguments_DuplicateShadowedNamesUseDeclarationIdentity(t *testing.T) {
	outerT := symbol.NewTypeParam("T", nil)
	innerT := symbol.NewTypeParam("T", nil)
	deepU := symbol.NewTypeParam("U", nil)

	receiver := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{outerT, innerT},
		DeclaredTypeParameters: []symbol.TypeParam{innerT},
	}
	deep := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{outerT, innerT, deepU},
		DeclaredTypeParameters: []symbol.TypeParam{deepU},
	}

	got := normalizeClassTypeArguments(
		deep,
		[]string{"Long"},
		receiver,
		[]string{"String", "Integer"},
	)
	want := []string{"String", "Integer", "Long"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Outer<T>.Inner<T>.Deep<U> arguments = %#v, want %#v", got, want)
	}

	allParameters := []symbol.TypeParam{outerT, innerT, deepU}
	symbol.DisambiguateTypeParamGoNames(allParameters)
	got = normalizeClassTypeArguments(deep, []string{"Long"}, receiver, nil)
	want = []string{"T", "T2", "Long"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("symbolic Outer<T>.Inner<T>.Deep<U> arguments = %#v, want %#v", got, want)
	}
}

func TestRawTypeParameterErasure_UsesBoundDeclarationIdentity(t *testing.T) {
	outerA := symbol.NewTypeParam("A", []symbol.JavaType{{Original: "OuterRoot"}})
	outerB := symbol.NewTypeParam("B", []symbol.JavaType{{Original: "A"}})
	outerParameters := []symbol.TypeParam{outerA, outerB}
	symbol.BindTypeParameterBounds(outerParameters[1:], outerParameters)

	innerA := symbol.NewTypeParam("A", []symbol.JavaType{{Original: "InnerRoot"}})
	visible := append(append([]symbol.TypeParam{}, outerParameters...), innerA)

	if got, want := rawTypeParameterErasure(outerParameters[1], visible), "OuterRoot"; got != want {
		t.Fatalf("B erasure = %q, want %q", got, want)
	}
}

func TestClassTypeArgumentStringsForScope_UsesCarriedDeclarationIdentity(t *testing.T) {
	outerT := symbol.NewTypeParam("T", nil)
	innerT := symbol.NewTypeParam("T", nil)
	outer := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{outerT},
		DeclaredTypeParameters: []symbol.TypeParam{outerT},
	}
	inner := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{outerT, innerT},
		DeclaredTypeParameters: []symbol.TypeParam{innerT},
	}

	got := classTypeArgumentStringsForScope(inner, []string{"String", "Integer"}, outer)
	want := []string{"String"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("enclosing arguments = %#v, want %#v", got, want)
	}
}

func TestDiamondClassTypeArgumentPrefix_CarriesOnlyEnclosingSlots(t *testing.T) {
	outerT := symbol.NewTypeParam("T", nil)
	innerU := symbol.NewTypeParam("U", nil)
	outer := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{outerT},
		DeclaredTypeParameters: []symbol.TypeParam{outerT},
	}
	inner := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{outerT, innerU},
		DeclaredTypeParameters: []symbol.TypeParam{innerU},
	}

	if got, want := diamondClassTypeArgumentPrefix(inner, outer, []string{"String"}), []string{"String"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("inner diamond prefix = %#v, want %#v", got, want)
	}
	if got := diamondClassTypeArgumentPrefix(outer, nil, nil); len(got) != 0 {
		t.Fatalf("top-level diamond prefix = %#v, want no explicit arguments", got)
	}
}

func TestObjectCreation_RawErasesWhileUntargetedDiamondRemainsInferable(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class DiamondRawDistinction<T> {
    T value;

    DiamondRawDistinction(T value) {
        this.value = value;
    }

    static void diamond() {
        new DiamondRawDistinction<>("diamond");
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    static void raw() {
        new DiamondRawDistinction("raw");
    }
}
`)
	if strings.Contains(out, "newDiamondRawDistinctionJava2goExecution[any](__java2goExecution, \"diamond\")") {
		t.Fatalf("untargeted diamond was incorrectly erased instead of inferred:\n%s", out)
	}
	if !strings.Contains(out, "newDiamondRawDistinctionJava2goExecution[any](__java2goExecution, \"raw\")") {
		t.Fatalf("raw construction did not explicitly use erased arguments:\n%s", out)
	}
}

func TestObjectCreation_QualifiedInnerPreservesOrderAndNullCheckTiming(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class QualifiedInnerOrderProbe {
    static int effects;
    static Outer saved;

    static class Outer {
        class Inner {
            Inner(int ignored) {
                effects = effects * 10 + 3;
            }
        }
    }

    static Outer qualifier() {
        effects = effects * 10 + 1;
        return saved;
    }

    static int argument() {
        effects = effects * 10 + 2;
        return 7;
    }

    public static String run() {
        saved = new Outer();
        effects = 0;
        qualifier().new Inner(argument());
        int ordered = effects;

        saved = null;
        effects = 0;
        try {
            qualifier().new Inner(argument());
            return "missing-npe";
        } catch (NullPointerException expected) {
            return ordered + ":" + effects;
        }
    }
}
`, "123:1")
}

func TestObjectCreation_UntargetedDiamondInfersPresentAndDefaultsMissingParameters(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class UntargetedDiamondProbe {
    static class Pair<A, B> {
        A first;
        B second;

        Pair(A first) {
            this.first = first;
        }
    }

    public static String run() {
        var pair = new Pair<>("left");
        pair.second = 7;
        return pair.first + ":" + pair.second;
    }
}
`, "left:7")
}

func TestObjectCreation_UnsupportedQualifierInferenceStillPassesEnclosingValue(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class SwitchQualifiedInnerProbe {
    static class Outer {
        String value;

        Outer(String value) {
            this.value = value;
        }

        class Inner {
            String read() {
                return value;
            }
        }
    }

    public static String run() {
        Outer left = new Outer("left");
        Outer right = new Outer("right");
        return (switch (1) {
            case 0 -> left;
            default -> right;
        }).new Inner().read();
    }
}
`, "right")
}

func TestInstantiatedConstructorParameterTypes_SubstitutesWildcardBounds(t *testing.T) {
	parameter := symbol.NewTypeParam("T", nil)
	scope := &symbol.ClassScope{
		TypeParameters:         []symbol.TypeParam{parameter},
		DeclaredTypeParameters: []symbol.TypeParam{parameter},
	}
	constructor := &symbol.Definition{Parameters: []*symbol.Definition{{
		OriginalType: "Supplier<? extends T>",
		TypeParameterBindings: map[string]*symbol.TypeParamDeclaration{
			"T": parameter.Declaration,
		},
	}}}

	got := instantiatedConstructorParameterTypes(constructor, scope, []string{"String"})
	want := []string{"Supplier<? extends String>"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("instantiated constructor parameters = %#v, want %#v", got, want)
	}
}

func TestMapClassTypeArgumentStringsToAncestor_MapsEachSuperclassEdge(t *testing.T) {
	helper := setupParseHelper(t, `
class GenericBase<A, B> {}
class GenericMiddle<X> extends GenericBase<String, X> {}
class GenericLeaf<Y> extends GenericMiddle<Y> {}
`)
	base := helper.File.Symbols.FindClassScope("GenericBase")
	leaf := helper.File.Symbols.FindClassScope("GenericLeaf")
	if base == nil || leaf == nil {
		t.Fatal("expected parsed base and leaf scopes")
	}

	got := mapClassTypeArgumentStringsToAncestor(leaf, []string{"Integer"}, base, helper.Ctx)
	want := []string{"String", "Integer"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("leaf superclass view = %#v, want %#v", got, want)
	}
}

func TestMapClassTypeArgumentStringsToAncestor_ShadowedOwnParameterWinsLexically(t *testing.T) {
	helper := setupParseHelper(t, `
class ShadowOuter<T> {
    class ShadowParent<P> {}
    class ShadowChild<T> extends ShadowParent<T> {}
}
`)
	parent := helper.File.Symbols.FindClassScope("ShadowParent")
	child := helper.File.Symbols.FindClassScope("ShadowChild")
	if parent == nil || child == nil {
		t.Fatal("expected parsed parent and child scopes")
	}

	got := mapClassTypeArgumentStringsToAncestor(child, []string{"String", "Integer"}, parent, helper.Ctx)
	want := []string{"String", "Integer"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shadowed superclass view = %#v, want %#v", got, want)
	}
}
