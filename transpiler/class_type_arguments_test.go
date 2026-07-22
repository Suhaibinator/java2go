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
