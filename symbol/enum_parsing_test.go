package symbol_test

import (
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
)

func TestParseSymbols_EnumInterfacesAndConstantBodies(t *testing.T) {
	src := `
package enums.symbols;
interface Flag { boolean isOn(); }
public enum Switch implements Flag {
    ON { public boolean isOn() { return true; } },
    OFF;
    public boolean isOn() { return false; }
}
`
	file := parsing.SourceFile{Name: "Switch.java", Source: []byte(src)}
	if err := file.ParseAST(); err != nil {
		t.Fatalf("failed to parse enum AST: %v", err)
	}

	symbols := file.ParseSymbols()
	base := symbols.FindClassScope("Switch")
	if base == nil {
		t.Fatalf("expected enum class scope to be populated")
	}

	if got := len(base.ImplementedInterfaces); got != 1 || base.ImplementedInterfaces[0] != "Flag" {
		t.Fatalf("expected implemented interfaces to include Flag, got: %#v", base.ImplementedInterfaces)
	}

	if got := len(base.EnumConstants); got != 2 {
		t.Fatalf("expected two enum constants, got %d", got)
	}
	if base.EnumConstants[0].Body == nil {
		t.Fatalf("expected enum constant with body to retain body node")
	}
	if base.EnumConstants[1].Body != nil {
		t.Fatalf("expected enum constant without body to have nil body node")
	}

	for _, required := range []string{"Name", "Ordinal", "CompareTo", "ValueOf"} {
		if len(base.FindMethod().ByName(required)) == 0 {
			t.Fatalf("expected synthetic method %s to be registered on enum", required)
		}
	}

	if len(base.FindMethod().ByOriginalName("isOn")) == 0 {
		t.Fatalf("expected user-declared method isOn to be registered")
	}
	if len(base.FindMethod().ByOriginalName("apply")) != 0 {
		t.Fatalf("did not expect unrelated synthetic methods to be present")
	}
}
