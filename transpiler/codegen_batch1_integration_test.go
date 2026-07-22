package transpiler

import (
	"strings"
	"testing"
)

// TestCodegen_TernaryUsesTypedLazyIIFE verifies a ternary lowers to a typed
// branch-local IIFE rather than an eager helper call.
func TestCodegen_TernaryUsesTypedLazyIIFE(t *testing.T) {
	src := `
public class T {
    public int pick(boolean b) {
        return b ? 1 : 2;
    }
}
`
	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "stdjava.Ternary(") {
		t.Errorf("ternary must not eagerly evaluate helper arguments, got:\n%s", out)
	}
	if !strings.Contains(out, "func() int32") || !strings.Contains(out, "if b") {
		t.Errorf("expected typed lazy ternary IIFE, got:\n%s", out)
	}
	if !strings.Contains(out, `stdjava "github.com/NickyBoy89/java2go/stdjava"`) ||
		!strings.Contains(out, "PickJava2goExecution(stdjava.NewExecution(), b)") {
		t.Errorf("expected the public ABI wrapper to create a Java execution, got:\n%s", out)
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
// `new T[]{...}` preserves its concrete element type through the Java array
// identity allocator.
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
	if !strings.Contains(flat, `stdjava.PrimitiveArrayLiteral[int32](stdjava.PrimitiveIntTypeID, 1, 2, 3)`) {
		t.Errorf("expected an int32 primitive array literal with the exact Java int descriptor, got:\n%s", out)
	}
	if !strings.Contains(flat, `stdjava.ReferenceArrayLiteralOf[string](stdjava.StringTypeID, "x", "y")`) {
		t.Errorf("expected a String[] literal with the exact Java String descriptor, got:\n%s", out)
	}
	if !strings.Contains(flat, `stdjava.ReferenceArrayLiteralOf[any](stdjava.ObjectTypeID, int32(1), "z")`) {
		t.Errorf("expected an Object[] literal with its descriptor and Java Integer-width boxing, got:\n%s", out)
	}
}

// TestCodegen_SizedArrayAllocationKeepsSliceType verifies every sized array
// carries exact reified Java component identity while retaining the concrete Go
// component type used by generated element operations.
func TestCodegen_SizedArrayAllocationKeepsSliceType(t *testing.T) {
	src := `
class Worker { int id; }
public class Alloc {
    public void run(int n) {
        int[] a = new int[n];
        String[] s = new String[n];
        Worker[] w = new Worker[n];
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, `stdjava.NewPrimitiveArray[int32](n, stdjava.PrimitiveIntTypeID)`) {
		t.Errorf("expected a reified int[] allocation with the exact primitive descriptor, got:\n%s", out)
	}
	if !strings.Contains(out, `stdjava.NewReferenceArrayOf[string](n, stdjava.StringTypeID)`) {
		t.Errorf("expected a reified String[] allocation with the exact Java String descriptor, got:\n%s", out)
	}
	// Worker is package-private, so its struct is lowercased to `worker`; the
	// element type must match.
	if !strings.Contains(out, `stdjava.NewReferenceArrayOf[*worker](n, stdjava.TypeID("Worker"))`) {
		t.Errorf("expected a reified Worker[] with the lowercased Go component type, got:\n%s", out)
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
	if !strings.Contains(flat, "dg.animal = newAnimalJava2goWithSelfJava2goExecution(__java2goExecution, __java2goMostDerived, name)") {
		t.Errorf("expected the super constructor call to preserve the most-derived receiver, got:\n%s", out)
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

// TestCodegen_LongShiftMaskWidth verifies long shifts mask the count to 6 bits
// while int shifts use 5 bits, matching Java.
func TestCodegen_LongShiftMaskWidth(t *testing.T) {
	src := `
public class LSh {
    public long bigLong() { return 1L << 32; }
    public int bigInt() { return 1 << 32; }
    public long overLong() { return 1L << 64; }
}
`
	out := renderGoFileFromJava(t, src)
	// long: 32 & 63 == 32, so the count is preserved.
	if !strings.Contains(out, "int64(1) << 32") {
		t.Errorf("expected long `1L << 32` to keep count 32 (6-bit mask), got:\n%s", out)
	}
	// int: 32 & 31 == 0.
	if !strings.Contains(out, "1 << 0") {
		t.Errorf("expected int `1 << 32` to mask to `1 << 0`, got:\n%s", out)
	}
	// long: 64 & 63 == 0.
	if !strings.Contains(out, "int64(1) << 0") {
		t.Errorf("expected long `1L << 64` to mask to count 0, got:\n%s", out)
	}
}

// TestCodegen_PackagePrivateConstructorCallCasing verifies `new T(...)` on a
// package-private class emits the lowercased constructor name.
func TestCodegen_PackagePrivateConstructorCallCasing(t *testing.T) {
	src := `
class Rectangle {
    Rectangle(double w, double h) {}
}
public class Maker {
    public void run() {
        Rectangle r = new Rectangle(2.0, 3.0);
    }
}
`
	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "Newrectangle") {
		t.Errorf("constructor call should be `newRectangle`, not miscased `Newrectangle`:\n%s", out)
	}
	if !strings.Contains(out, "newRectangleJava2goExecution(__java2goExecution, 2.0, 3.0)") {
		t.Errorf("expected execution-aware `newRectangle(2.0, 3.0)`, got:\n%s", out)
	}
}

// TestCodegen_PackagePrivateInterfaceCasing verifies a package-private interface
// is embedded and referenced by its lowercased generated name (and by value).
func TestCodegen_PackagePrivateInterfaceCasing(t *testing.T) {
	src := `
interface Greeter {
    String greet(String who);
}
class Formal implements Greeter {
    public String greet(String who) { return "Hi " + who; }
}
public class App {
    public void run() {
        Greeter[] gs = new Greeter[] { new Formal() };
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(out, "type greeter interface") {
		t.Errorf("expected lowercased `type greeter interface`, got:\n%s", out)
	}
	if strings.Contains(flat, "type formal struct { Greeter") {
		t.Errorf("interface embed should be lowercased `greeter`, got:\n%s", out)
	}
	if !strings.Contains(flat, "type formal struct { greeter") {
		t.Errorf("expected struct to embed `greeter`, got:\n%s", out)
	}
	// Interface element type is by value, not a pointer, and the literal retains
	// the Java interface component descriptor for checked covariant stores.
	if !strings.Contains(flat, `stdjava.ReferenceArrayLiteralOf[greeter](stdjava.TypeID("Greeter"), newFormalJava2goExecution(__java2goExecution))`) {
		t.Errorf("expected a reified greeter[] literal with an interface value element, got:\n%s", out)
	}
}

// TestCodegen_EnumMethodCallCasing verifies a built-in enum method call resolves
// to the generated (capitalized) Go method name.
func TestCodegen_EnumMethodCallCasing(t *testing.T) {
	src := `
public class E {
    enum Day { MON, TUE, WED }
    public int wedOrdinal() {
        return Day.WED.ordinal();
    }
}
`
	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, ".ordinal()") {
		t.Errorf("enum method call should resolve to `.Ordinal()`, got lowercased:\n%s", out)
	}
	if !strings.Contains(out, "WED.Ordinal()") {
		t.Errorf("expected `WED.Ordinal()`, got:\n%s", out)
	}
}
