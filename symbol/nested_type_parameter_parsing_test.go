package symbol_test

import (
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func TestParseSymbols_SeparatesNestedDeclaredAndCarriedTypeParameters(t *testing.T) {
	src := `
class Outer<T> {
    class Inner<U> {}
    class Leaf {}
}
`
	file := parsing.SourceFile{Name: "Outer.java", Source: []byte(src)}
	if err := file.ParseAST(); err != nil {
		t.Fatalf("failed to parse AST: %v", err)
	}

	symbols := file.ParseSymbols()
	outer := symbols.FindClassScope("Outer")
	if outer == nil || len(outer.Subclasses) != 2 {
		t.Fatalf("expected Outer with two nested classes, got %#v", outer)
	}
	inner := outer.Subclasses[0]
	if got, want := inner.TypeParameterNames(), []string{"T", "U"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Inner carried parameters = %#v, want %#v", got, want)
	}
	if got, want := symbol.TypeParamNames(inner.DeclaredTypeParameters), []string{"U"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Inner declared parameters = %#v, want %#v", got, want)
	}

	leaf := outer.Subclasses[1]
	if got, want := leaf.TypeParameterNames(), []string{"T"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Leaf carried parameters = %#v, want %#v", got, want)
	}
	if leaf.DeclaredTypeParameters == nil || len(leaf.OwnTypeParameters()) != 0 {
		t.Fatalf("Leaf must record an explicit empty declared-parameter set, got %#v", leaf.DeclaredTypeParameters)
	}
}
