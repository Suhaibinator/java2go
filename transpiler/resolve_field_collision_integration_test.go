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

func TestResolveFile_StaticMethodHidingGetsPackageUniqueNames(t *testing.T) {
	src := `
class StaticParent {
    static String kind() { return "parent"; }
}
class StaticChild extends StaticParent {
    static String kind() { return "child"; }
}
class InstanceOne {
    String kind() { return "one"; }
}
class InstanceTwo {
    String kind() { return "two"; }
}
public class StaticHidingProgram {
    static String run() {
        return StaticParent.kind() + ":" + StaticChild.kind() + ":"
            + new InstanceOne().kind() + ":" + new InstanceTwo().kind();
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "func kind0() string") || !strings.Contains(flat, "func kind() string") {
		t.Fatalf("hidden static methods did not receive distinct package names:\n%s", out)
	}
	if strings.Count(flat, "func (ie *instanceOne) kind() string") != 1 ||
		strings.Count(flat, "func (io *instanceTwo) kind() string") != 1 {
		t.Fatalf("instance methods in distinct method sets were unnecessarily renamed:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestStaticHidingRuntime(t *testing.T) {
	if got := run(); got != "parent:child:one:two" {
		t.Fatalf("run() = %q", got)
	}
}
`)
}
