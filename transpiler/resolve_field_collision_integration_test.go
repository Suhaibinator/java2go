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

func TestResolveFile_FieldMethodNamespacesRemainDistinctAtRuntime(t *testing.T) {
	src := `
class InstanceMemberCollision {
    int value;

    InstanceMemberCollision(int value) {
        this.value = value;
    }

    int value() {
        return value;
    }

    int value(int increment) {
        return value + increment;
    }
}

class StaticSameClassCollision {
    static int score = 5;

    static int score() {
        return score + 1;
    }

    static int evaluate() {
        return score * 10 + score();
    }
}

class StaticFieldOwner {
    static int token = 3;

    static int read() {
        return token;
    }
}

class StaticMethodOwner {
    static int token() {
        return 4;
    }
}

public class MemberCollisionProgram {
    public static String run() {
        InstanceMemberCollision value = new InstanceMemberCollision(7);
        return value.value + ":" + value.value() + ":" + value.value(1)
            + ":" + StaticSameClassCollision.evaluate()
            + ":" + StaticFieldOwner.read() + ":" + StaticMethodOwner.token();
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestMemberNamespaceRuntime(t *testing.T) {
	if got := Run(); got != "7:7:8:56:3:4" {
		t.Fatalf("Run() = %q, want 7:7:8:56:3:4", got)
	}
}
`)
}

func TestResolveFile_PromotedFieldMethodNamespacesRemainDistinctAtRuntime(t *testing.T) {
	src := `
class MethodChild extends FieldBase {
    int value() {
        return 4;
    }

    int observe() {
        return value * 10 + value();
    }
}

class FieldChild extends MethodBase {
    int score = 6;

    int observe() {
        return score * 10 + score();
    }
}

class FieldBase {
    int value = 3;
}

class MethodBase {
    int score() {
        return 5;
    }
}

public class PromotedMemberCollisionProgram {
    public static String run() {
        return new MethodChild().observe() + ":" + new FieldChild().observe();
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestPromotedMemberNamespaceRuntime(t *testing.T) {
	if got := Run(); got != "34:65" {
		t.Fatalf("Run() = %q, want 34:65", got)
	}
}
`)
}

func TestResolveFile_InterfaceDefaultsAndCarriersRemainDistinctAtRuntime(t *testing.T) {
	src := `
interface DefaultValue {
    default int value() {
        return 4;
    }
}

class DefaultFieldCollision implements DefaultValue {
    int value = 3;

    int observe() {
        return value * 10 + value();
    }
}

interface Token {
    default int read() {
        return 2;
    }
}

class CarrierNameCollision implements Token {
    int tokenDefaults = 3;

    int tokenDefaults() {
        return 4;
    }

    String observe() {
        return tokenDefaults + ":" + tokenDefaults() + ":" + read();
    }
}

public class InterfaceMemberCollisionProgram {
    public static String run() {
        return new DefaultFieldCollision().observe() + ":"
            + new CarrierNameCollision().observe();
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestInterfaceMemberNamespaceRuntime(t *testing.T) {
	if got := Run(); got != "34:3:4:2" {
		t.Fatalf("Run() = %q, want 34:3:4:2", got)
	}
}
`)
}
