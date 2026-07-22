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
}
`
	file := parsing.SourceFile{Name: "Outer.java", Source: []byte(src)}
	if err := file.ParseAST(); err != nil {
		t.Fatalf("failed to parse AST: %v", err)
	}

	symbols := file.ParseSymbols()
	outer := symbols.FindClassScope("Outer")
	if outer == nil || len(outer.Subclasses) != 1 {
		t.Fatalf("expected Outer with one nested class, got %#v", outer)
	}
	inner := outer.Subclasses[0]
	if got, want := inner.TypeParameterNames(), []string{"T", "U"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Inner carried parameters = %#v, want %#v", got, want)
	}
	if got, want := symbol.TypeParamNames(inner.DeclaredTypeParameters), []string{"U"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Inner declared parameters = %#v, want %#v", got, want)
	}
}
