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

	if !strings.Contains(flat, "type Cat struct { *Animal Pet }") {
		t.Fatalf("expected Cat to embed superclass and interfaces, got:\n%s", out)
	}
	if strings.Contains(flat, "*Pet }") {
		t.Fatalf("expected Cat to embed interfaces without pointer indirection, got:\n%s", out)
	}
}

func TestSuperclassMethodResolution(t *testing.T) {
	src := `
	package inherit;
	public class Animal {
	    public void speak() {}
	}
	public class Cat extends Animal {
	    public void test() { this.speak(); }
	}
	`

	helper := setupParseHelper(t, src)
	catScope := helper.File.Symbols.FindClassScope("Cat")
	if catScope == nil {
		t.Fatalf("expected to find Cat scope in symbols")
	}
	if strings.TrimSpace(catScope.Superclass) == "" {
		t.Fatalf("expected Cat to record superclass in symbols, got empty")
	}
	ctx := helper.Ctx
	ctx.currentClass = catScope
	if scope := resolveClassScopeByQualifiedName(ctx, "Animal"); scope == nil {
		t.Fatalf("expected resolveClassScopeByQualifiedName to find Animal in current file symbols")
	}
	if resolved := findInstanceMethodInHierarchy(catScope, "speak", 0, ctx); resolved == nil || resolved.def == nil || resolved.def.Name != "Speak" {
		t.Fatalf("expected hierarchy method resolution to find Animal.speak as Speak, got: %#v", resolved)
	}

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "ct.Speak()") {
		t.Fatalf("expected inherited method call to resolve to exported Go name, got:\n%s", out)
	}
}
