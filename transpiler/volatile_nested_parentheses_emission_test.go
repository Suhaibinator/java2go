package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

// Artifact capture only: preserve unmodified compiler output before the JDK/Go
// behavior witness fails, so temporary-module cleanup cannot erase the cause.
func TestVolatileNestedParenthesesFrozenEmission(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "volatile_nested_parentheses", "VolatileNestedParenthesesProbe.java"))
	if err != nil {
		t.Fatal(err)
	}
	generated := renderGoFileFromJava(t, string(source))
	dir := os.Getenv("JAVA2GO_NESTED_PARENTHESES_EMISSION_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	file, err := os.OpenFile(filepath.Join(dir, "generated.go"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.WriteString(generated); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			t.Errorf("close failed emission file: %v", closeErr)
		}
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
}
