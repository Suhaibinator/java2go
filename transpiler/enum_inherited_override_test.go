package transpiler

import (
	"strings"
	"testing"
)

// Java permits a constant body to override Enum.toString without redeclaring
// that method in the enum. The inherited default still belongs to other values.
func TestEnumInheritedToStringConstantBodyPlanning(t *testing.T) {
	out := renderGoFileFromJava(t, `
public enum ConstantOnly {
    A { public String toString() { String local = name(); return local; } },
    B;
}
`)
	if !strings.Contains(out, "_ConstantOnly_A_") {
		t.Fatalf("constant-only Enum.toString override was not lowered:\n%s", out)
	}
	if !strings.Contains(out, "return local") {
		t.Fatalf("constant callback body was not retained:\n%s", out)
	}
	if !strings.Contains(out, "stdjava.EnumNameJavaString(") {
		t.Fatalf("inherited Enum.toString must return the stored name reference:\n%s", out)
	}
}

func TestEnumInheritedToStringIgnoresConstantOverloadsAndNestedMethods(t *testing.T) {
	out := renderGoFileFromJava(t, `
public enum OnlyOverloads {
    A {
        public String toString(int ignored) { return "overload"; }
        class Nested { public String toString() { return "nested"; } }
    },
    B;
}
`)
	if strings.Contains(out, "_OnlyOverloads_A_") {
		t.Fatalf("an overload or nested member was mistaken for Enum.toString:\n%s", out)
	}
}
