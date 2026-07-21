package transpiler

import (
	"strings"
	"testing"
)

func TestResolveFile_InstanceFieldDoesNotConflictWithOtherClassStaticField(t *testing.T) {
	src := `
class A {
    int x;
    int x0;
}
class B {
    static int x;
}
`
	helper := setupParseHelper(t, src)
	classA := helper.File.Symbols.FindClassScope("A")
	classB := helper.File.Symbols.FindClassScope("B")
	if classA == nil || classB == nil {
		t.Fatalf("parsed classes = (%v, %v), want A and B", classA, classB)
	}
	if got := classA.FindFieldByName("x").Name; got != "x" {
		t.Fatalf("A.x resolved name = %q, want x", got)
	}
	if got := classA.FindFieldByName("x0").Name; got != "x0" {
		t.Fatalf("A.x0 resolved name = %q, want x0", got)
	}
	if got := classB.FindFieldByName("x").Name; got != "x" {
		t.Fatalf("B.x resolved name = %q, want x", got)
	}

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "type a struct { x int32 x0 int32 }") {
		t.Fatalf("generated A fields were renamed or duplicated:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestResolvedFieldsCompile(t *testing.T) {
    value := &a{x: 1, x0: 2}
    if value.x + value.x0 != 3 {
        t.Fatal("unexpected field values")
    }
}
`)
}
