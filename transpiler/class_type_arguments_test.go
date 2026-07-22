package transpiler

import (
	"reflect"
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
