package symbol_test

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func TestTypeParameterNominalReservationRetainsBindingsAndAliases(t *testing.T) {
	outer := symbol.NewTypeParam("Item", nil)
	inner := symbol.NewTypeParam("Item", []symbol.JavaType{{Original: "probe.Item", TypeParameterBindings: map[string]*symbol.TypeParamDeclaration{}}})
	parameters := []symbol.TypeParam{outer, inner}
	symbol.DisambiguateTypeParamGoNames(parameters)
	copyOfOuter := outer
	sourceBound := inner.Bounds[0].Original
	nominal := map[string]struct{}{"Item": {}, "Item3": {}}
	symbol.ReserveTypeParamNominalNames(parameters, nominal)
	if outer.EmittedName() != "Item4" || inner.EmittedName() != "Item2" {
		t.Fatalf("names = %q, %q", outer.EmittedName(), inner.EmittedName())
	}
	if outer.Name != "Item" || inner.Name != "Item" || copyOfOuter.Declaration != outer.Declaration || copyOfOuter.EmittedName() != "Item4" || inner.Bounds[0].Original != sourceBound {
		t.Fatal("source declaration identity or nominal bound changed")
	}
	symbol.ReserveTypeParamNominalNames(parameters, nominal)
	later := symbol.NewTypeParam("Item", nil)
	symbol.DisambiguateTypeParamGoNames(symbol.AppendTypeParamsByDeclaration(parameters, []symbol.TypeParam{later}))
	if outer.EmittedName() != "Item4" || inner.EmittedName() != "Item2" {
		t.Fatal("later context changed previously emitted names")
	}
}

func TestTypeParameterNominalReservationKeepsIndependentMethodNames(t *testing.T) {
	first := symbol.NewTypeParam("T", nil)
	second := symbol.NewTypeParam("T", nil)
	symbol.ReserveTypeParamNominalNames([]symbol.TypeParam{first, second}, map[string]struct{}{"Other": {}})
	if first.EmittedName() != "T" || second.EmittedName() != "T" {
		t.Fatal("unrelated method binders were needlessly renamed")
	}
}
