package transpiler

import (
	"strings"
	"testing"
)

// TestCodegen_TernaryQualifiedWithImport verifies a ternary lowers to a
// stdjava-qualified Ternary call and pulls in the stdjava import.
func TestCodegen_TernaryQualifiedWithImport(t *testing.T) {
	src := `
public class T {
    public int pick(boolean b) {
        return b ? 1 : 2;
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "stdjava.Ternary(") {
		t.Errorf("expected stdjava.Ternary call, got:\n%s", out)
	}
	if strings.Contains(out, "ternary(") && !strings.Contains(out, "stdjava.Ternary(") {
		t.Errorf("expected qualified ternary, got unqualified:\n%s", out)
	}
	if !strings.Contains(out, "stdjava") || !strings.Contains(out, "import") {
		t.Errorf("expected stdjava import to be added, got:\n%s", out)
	}
}

// TestCodegen_DoWhileNegatesCondition verifies the do-while break condition is a
// proper logical negation rather than an ILLEGAL unary operator.
func TestCodegen_DoWhileNegatesCondition(t *testing.T) {
	src := `
public class D {
    public void run() {
        int j = 0;
        do {
            j++;
        } while (j < 2);
    }
}
`
	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "ILLEGAL") {
		t.Errorf("do-while emitted ILLEGAL operator, got:\n%s", out)
	}
	if !strings.Contains(out, "if !(j < 2)") {
		t.Errorf("expected negated break condition `if !(j < 2)`, got:\n%s", out)
	}
}

// TestCodegen_ClassicSwitchLowering verifies a classic switch statement lowers to
// a Go switch with grouped case labels and break-by-default semantics (the
// explicit Java `break` is dropped).
func TestCodegen_ClassicSwitchLowering(t *testing.T) {
	src := `
public class S {
    public void run(int k) {
        switch (k) {
            case 0:
                System.out.println("a");
                break;
            case 1:
            case 2:
                System.out.println("b");
                break;
            default:
                System.out.println("c");
        }
    }
}
`
	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "UNSUPPORTED") {
		t.Errorf("classic switch emitted an UNSUPPORTED stub, got:\n%s", out)
	}
	if !strings.Contains(out, "switch k") {
		t.Errorf("expected `switch k`, got:\n%s", out)
	}
	if !strings.Contains(normalizeSpaces(out), "case 1, 2:") {
		t.Errorf("expected stacked labels to merge into `case 1, 2:`, got:\n%s", out)
	}
	if !strings.Contains(out, "default:") {
		t.Errorf("expected a default clause, got:\n%s", out)
	}
}

// TestCodegen_SwitchFallthrough verifies a case that does not break gets an
// explicit Go `fallthrough`.
func TestCodegen_SwitchFallthrough(t *testing.T) {
	src := `
public class F {
    public void run(int k) {
        switch (k) {
            case 0:
                System.out.println("a");
            case 1:
                System.out.println("b");
                break;
        }
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "fallthrough") {
		t.Errorf("expected explicit fallthrough for a non-breaking case, got:\n%s", out)
	}
}

// TestCodegen_UpdateExpressionInExpressionPosition verifies ++/-- used as an
// expression route through stdjava pointer helpers.
func TestCodegen_UpdateExpressionInExpressionPosition(t *testing.T) {
	src := `
public class U {
    public void run() {
        int c = 0;
        System.out.println(c++);
        System.out.println(++c);
        int[] arr = {1, 2, 3};
        int i = 0;
        System.out.println(arr[i++]);
    }
}
`
	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "PostUpdate(") || strings.Contains(out, "PreUpdate(") {
		t.Errorf("expected stdjava increment helpers, got undefined PostUpdate/PreUpdate:\n%s", out)
	}
	if !strings.Contains(out, "stdjava.PostIncrement(&c)") {
		t.Errorf("expected stdjava.PostIncrement(&c), got:\n%s", out)
	}
	if !strings.Contains(out, "stdjava.PreIncrement(&c)") {
		t.Errorf("expected stdjava.PreIncrement(&c), got:\n%s", out)
	}
	if !strings.Contains(out, "stdjava.PostIncrement(&i)") {
		t.Errorf("expected stdjava.PostIncrement(&i) for arr[i++], got:\n%s", out)
	}
}

// TestCodegen_ArrayCreationWithInitializerKeepsType verifies that
// `new T[]{...}` emits a typed composite literal instead of a bare `{...}`.
func TestCodegen_ArrayCreationWithInitializerKeepsType(t *testing.T) {
	src := `
public class A {
    public void run() {
        int[] a = new int[]{1, 2, 3};
        String[] s = new String[]{"x", "y"};
        Object[] o = new Object[]{1, "z"};
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "[]int32{1, 2, 3}") {
		t.Errorf("expected []int32{1, 2, 3}, got:\n%s", out)
	}
	if !strings.Contains(flat, `[]string{"x", "y"}`) {
		t.Errorf("expected []string{...}, got:\n%s", out)
	}
	if !strings.Contains(flat, `[]any{1, "z"}`) {
		t.Errorf("expected []any{...} for Object[], got:\n%s", out)
	}
}

// TestCodegen_ShiftCountMasking verifies constant shift counts are masked at
// transpile time to match Java (int shifts mask to 5 bits) and that `>>>` is
// stdjava-qualified.
func TestCodegen_ShiftCountMasking(t *testing.T) {
	src := `
public class Sh {
    public int big() { return 1 << 32; }
    public int small() { return 1 << 1; }
    public int unsigned(int v) { return v >>> 1; }
    public int variable(int n) { return 1 << n; }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "1 << 0") {
		t.Errorf("expected `1 << 32` to mask to `1 << 0`, got:\n%s", out)
	}
	if !strings.Contains(out, "1 << 1") {
		t.Errorf("expected in-range `1 << 1` to be preserved, got:\n%s", out)
	}
	if !strings.Contains(out, "stdjava.UnsignedRightShift(") {
		t.Errorf("expected stdjava.UnsignedRightShift for >>>, got:\n%s", out)
	}
	if !strings.Contains(out, "1 << n") {
		t.Errorf("expected variable shift `1 << n` to be left as-is, got:\n%s", out)
	}
}

// TestCodegen_PackagePrivateInheritanceCasing verifies that a package-private
// superclass is embedded and constructed using its generated (lowercased) Go
// names, so the generated code compiles.
func TestCodegen_PackagePrivateInheritanceCasing(t *testing.T) {
	src := `
class Animal {
    String name;
    Animal(String name) { this.name = name; }
    String speak() { return "..."; }
}
class Dog extends Animal {
    Dog(String name) { super(name); }
    String speak() { return "Woof"; }
}
public class AnimalMain {
    public static void main(String[] args) {}
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	// The superclass struct is lowercased; the embed and constructor call must use
	// the same casing.
	if !strings.Contains(out, "type animal struct") {
		t.Errorf("expected lowercased `type animal struct`, got:\n%s", out)
	}
	if !strings.Contains(flat, "type dog struct { *animal") {
		t.Errorf("expected embed of `*animal` (lowercased), got:\n%s", out)
	}
	if !strings.Contains(out, "newAnimal(name)") {
		t.Errorf("expected super constructor call `newAnimal(name)`, got:\n%s", out)
	}
	if strings.Contains(out, "Newanimal") {
		t.Errorf("super constructor call should be `newAnimal`, not `Newanimal`, got:\n%s", out)
	}
}

// TestCodegen_PackagePrivateInheritance_RuntimeBehavior verifies the generated
// code for package-private inheritance compiles and dispatches the override.
func TestCodegen_PackagePrivateInheritance_RuntimeBehavior(t *testing.T) {
	src := `
class Animal {
    String name;
    Animal(String name) { this.name = name; }
    String speak() { return "..."; }
}
class Dog extends Animal {
    Dog(String name) { super(name); }
    String speak() { return "Woof"; }
}
public class AnimalApp {
    public static String run() {
        Dog d = new Dog("Rex");
        return d.speak();
    }
}
`
	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestPkgPrivateInheritance(t *testing.T) {
	if got := Run(); got != "Woof" {
		t.Fatalf("Run() = %q, want \"Woof\"", got)
	}
}
`)
}

// TestCodegen_QualifiedEnumConstantAccess verifies `Enum.CONSTANT` in expression
// position lowers to the bare generated constant var.
func TestCodegen_QualifiedEnumConstantAccess(t *testing.T) {
	src := `
public class E {
    enum Day { MON, TUE, WED }
    public Day pick() {
        return Day.WED;
    }
    public boolean isWed(Day d) {
        return d == Day.WED;
    }
}
`
	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "Day.WED") {
		t.Errorf("expected `Day.WED` to lower to the constant var, got verbatim selector:\n%s", out)
	}
	if !strings.Contains(out, "return WED") {
		t.Errorf("expected `return WED`, got:\n%s", out)
	}
	if !strings.Contains(out, "== WED") {
		t.Errorf("expected `d == WED`, got:\n%s", out)
	}
}
