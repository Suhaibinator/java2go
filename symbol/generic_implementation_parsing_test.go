package symbol_test

import (
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
)

func TestParseSymbols_PreservesImplementedInterfaceTypeArguments(t *testing.T) {
	src := `
package generic.symbols;
interface TaskRule<T> {}
class Event {}
class GenericRule<T> implements TaskRule<T> {}
class EventRule implements TaskRule<Event> {}
`
	file := parsing.SourceFile{Name: "Rules.java", Source: []byte(src)}
	if err := file.ParseAST(); err != nil {
		t.Fatalf("failed to parse AST: %v", err)
	}

	symbols := file.ParseSymbols()
	tests := []struct {
		className string
		want      []string
	}{
		{className: "GenericRule", want: []string{"TaskRule<T>"}},
		{className: "EventRule", want: []string{"TaskRule<Event>"}},
	}
	for _, test := range tests {
		scope := symbols.FindClassScope(test.className)
		if scope == nil {
			t.Fatalf("expected class scope for %s", test.className)
		}
		if !reflect.DeepEqual(scope.ImplementedInterfaces, test.want) {
			t.Fatalf("%s implemented interfaces = %#v, want %#v", test.className, scope.ImplementedInterfaces, test.want)
		}
	}
}
