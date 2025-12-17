package main

import (
	"strings"
	"testing"
)

func TestClassExtendsAndImplementsEmbeddedFields(t *testing.T) {
	src := `
package inherit;
public class Cat extends Animal implements Pet {
    public void pat() {}
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "type Cat struct { *Animal *Pet }") {
		t.Fatalf("expected Cat to embed superclass and interfaces, got:\n%s", out)
	}
}
