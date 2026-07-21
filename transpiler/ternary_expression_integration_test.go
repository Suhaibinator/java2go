package transpiler

import (
	"strings"
	"testing"
)

func TestTernaryExpressionLazyRuntimeAndResultTyping(t *testing.T) {
	src := `
class TernaryReference {
    private int value;

    TernaryReference(int value) {
        this.value = value;
    }

    public int getValue() {
        return this.value;
    }
}

public class TernaryExpressionProgram {
    private static int effects = 0;

    private static int mark(int value) {
        effects = effects * 10 + value;
        return value;
    }

    private static int outOfBounds() {
        int[] values = new int[] { 4 };
        return values[9];
    }

    public static int lazyEvaluation() {
        effects = 0;
        int first = true ? mark(1) : mark(9);
        int second = false ? mark(8) : mark(2);
        int safe = true ? 7 : outOfBounds();
        return effects * 1000 + first * 100 + second * 10 + safe;
    }

    public static long primitive(boolean choose, int value) {
        return choose ? value : 5000000000L;
    }

    public static byte inferredNarrow(boolean choose) {
        byte value = 5;
        var selected = choose ? value : 7;
        return selected;
    }

    public static long nested(boolean first, boolean second) {
        var selected = first ? (second ? 1 : 2L) : 3L;
        return selected;
    }

    public static TernaryReference reference(boolean choose) {
        var selected = choose ? new TernaryReference(42) : null;
        return selected;
    }

    public static Object objectChoice(boolean choose) {
        return choose ? 7 : "seven";
    }

    public static Object inferredNullablePrimitive(boolean choose) {
        var selected = choose ? null : 11;
        return selected;
    }

    public static void main(String[] args) {
        System.out.println(lazyEvaluation());
        System.out.println(primitive(true, 9));
        System.out.println(primitive(false, 9));
        System.out.println(inferredNarrow(true));
        System.out.println(inferredNarrow(false));
        System.out.println(reference(true).getValue());
        System.out.println(reference(false) == null);
        System.out.println(objectChoice(true));
        System.out.println(objectChoice(false));
        System.out.println(inferredNullablePrimitive(true) == null);
        System.out.println(inferredNullablePrimitive(false));
        System.out.println(nested(true, false));
        System.out.println(nested(false, true));
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "stdjava.Ternary") {
		t.Fatalf("ternary lowering must not use the eager runtime helper:\n%s", out)
	}
	for _, typedIIFE := range []string{"func() int32", "func() int64", "func() int8", "func() *ternaryReference", "func() any"} {
		if !strings.Contains(out, typedIIFE) {
			t.Fatalf("expected generated ternary result type %q:\n%s", typedIIFE, out)
		}
	}
	runGoTestInTempModule(t, out, `
package main

import (
    "io"
    "os"
    "testing"
)

func TestTernaryTypesAndExactOutput(t *testing.T) {
    var _ int64 = Primitive(true, int32(9))
    var _ int8 = InferredNarrow(true)
    if _, ok := ObjectChoice(true).(int32); !ok {
        t.Fatalf("ObjectChoice(true) has type %T, want Java Integer/int32", ObjectChoice(true))
    }
    if _, ok := InferredNullablePrimitive(false).(int32); !ok {
        t.Fatalf("InferredNullablePrimitive(false) has type %T, want Java Integer/int32", InferredNullablePrimitive(false))
    }

    original := os.Stdout
    reader, writer, err := os.Pipe()
    if err != nil {
        t.Fatal(err)
    }
    os.Stdout = writer
    Main()
    if err := writer.Close(); err != nil {
        t.Fatal(err)
    }
    os.Stdout = original

    output, err := io.ReadAll(reader)
    if err != nil {
        t.Fatal(err)
    }
    const want = "12127\n9\n5000000000\n5\n7\n42\ntrue\n7\nseven\ntrue\n11\n2\n3\n"
    if got := string(output); got != want {
        t.Fatalf("Main output = %q, want %q", got, want)
    }
}
`)
}

func TestTernaryIgnoresIncompatibleInheritedBooleanTarget(t *testing.T) {
	src := `
public class NestedTernaryContextProgram {
    public static boolean run(int value) {
        return ((true ? value : -100) == value)
                && (((false ? 100 : 984) / 3) > value);
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "func() bool") {
		t.Fatalf("numeric ternary inherited enclosing boolean target:\n%s", out)
	}
	if got := strings.Count(out, "func() int32"); got != 2 {
		t.Fatalf("numeric ternary IIFE count = %d, want 2:\n%s", got, out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestNestedNumericTernaries(t *testing.T) {
    if !Run(int32(-2)) {
        t.Fatal("Run(-2) = false, want true")
    }
}
`)
}

func TestNullableValueBackedTernaryLocals(t *testing.T) {
	src := `
public class NullableTernaryLocalProgram {
    public static String stringCase(boolean choose) {
        String value = choose ? null : "go";
        boolean wasNull = value == null;
        if (wasNull) {
            value = "fallback";
        }
        String concatenated = value + "!";
        int length = value.length();
        value = value.concat("?");
        return wasNull + ":" + concatenated + ":" + length + ":" + value;
    }

    public static String boxedCase(boolean choose) {
        Integer value = choose ? null : 7;
        boolean wasNull = value == null;
        if (wasNull) {
            value = 11;
        }
        value += 1;
        return wasNull + ":" + value;
    }

    public static Integer boxedReturn(boolean choose) {
        Integer value = choose ? null : 7;
        if (value == null) {
            value = 13;
        }
        return value;
    }
}
`

	out := renderGoFileFromJava(t, src)
	if got := strings.Count(out, "var value any"); got < 3 {
		t.Fatalf("nullable value-backed ternary locals using interface storage = %d, want at least 3:\n%s", got, out)
	}
	if !strings.Contains(out, "stdjava.StringRequireNonNull(value)") {
		t.Fatalf("nullable String receiver was not normalized before length/concat:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestNullableTernaryLocals(t *testing.T) {
    stringCases := []struct {
        choose bool
        want string
    }{
        {true, "true:fallback!:8:fallback?"},
        {false, "false:go!:2:go?"},
    }
    for _, test := range stringCases {
        if got := StringCase(test.choose); got != test.want {
            t.Fatalf("StringCase(%v) = %q, want %q", test.choose, got, test.want)
        }
    }

    boxedCases := []struct {
        choose bool
        want string
    }{
        {true, "true:12"},
        {false, "false:8"},
    }
    for _, test := range boxedCases {
        if got := BoxedCase(test.choose); got != test.want {
            t.Fatalf("BoxedCase(%v) = %q, want %q", test.choose, got, test.want)
        }
    }
    if got := BoxedReturn(true); got != 13 {
        t.Fatalf("BoxedReturn(true) = %d, want 13", got)
    }
    if got := BoxedReturn(false); got != 7 {
        t.Fatalf("BoxedReturn(false) = %d, want 7", got)
    }
}
`)
}

func TestTernaryStandaloneTypingOverloadsBoxingConstantsAndArrays(t *testing.T) {
	src := `
public class TernaryTypingProgram {
    private static String choose(long value) { return "L"; }
    private static String choose(Object value) { return "O"; }
    private static String pick(Object value) { return "O"; }
    private static String pick(String value) { return "S"; }

    public static String nestedOverload(boolean first) {
        return choose(first ? 1 : 2L);
    }

    public static String nullOverload(boolean first) {
        return pick(first ? null : null);
    }

    public static boolean numericBox(boolean first) {
        Object selected = first ? 1 : 2L;
        return selected instanceof Long;
    }

    public static double floatPrecision(boolean first) {
        int integer = 16777217;
        float floating = 0.0F;
        return first ? integer : floating;
    }

    public static byte hexNarrow(boolean first) {
        byte narrow = 1;
        var selected = first ? narrow : 0x7f;
        return selected;
    }

    public static byte octalNarrow(boolean first) {
        byte narrow = 1;
        var selected = first ? narrow : 0177;
        return selected;
    }

    public static byte binaryNarrow(boolean first) {
        byte narrow = 1;
        var selected = first ? narrow : 0b01111111;
        return selected;
    }

    public static byte expressionNarrow(boolean first) {
        byte narrow = 1;
        var selected = first ? narrow : (0x70 | 0x0f);
        return selected;
    }

    public static int arrayLUB(boolean first) {
        var selected = first ? new String[2] : new Object[3];
        return selected.length;
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "choose(func() any") {
		t.Fatalf("numeric conditional used enclosing Object return as its overload type:\n%s", out)
	}
	if !strings.Contains(out, "func() int64") {
		t.Fatalf("numeric conditional did not retain standalone long promotion:\n%s", out)
	}
	if !strings.Contains(out, "func() []any") {
		t.Fatalf("reference-array conditional did not retain Object[] LUB:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestTernaryStandaloneTyping(t *testing.T) {
    if got := NestedOverload(true); got != "L" {
        t.Fatalf("NestedOverload(true) = %q, want L", got)
    }
    if got := NullOverload(true); got != "S" {
        t.Fatalf("NullOverload(true) = %q, want S", got)
    }
    if !NumericBox(true) || !NumericBox(false) {
        t.Fatalf("NumericBox(true)=%v NumericBox(false)=%v, want true/true", NumericBox(true), NumericBox(false))
    }
    if got := FloatPrecision(true); got != 16777216 {
        t.Fatalf("FloatPrecision(true) = %v, want float32-rounded 16777216", got)
    }
    for name, got := range map[string]int8{
        "hex": HexNarrow(false),
        "octal": OctalNarrow(false),
        "binary": BinaryNarrow(false),
        "expression": ExpressionNarrow(false),
    } {
        if got != 127 {
            t.Fatalf("%s narrowing = %d, want 127", name, got)
        }
    }
    if got := ArrayLUB(true); got != 2 {
        t.Fatalf("ArrayLUB(true) = %d, want 2", got)
    }
    if got := ArrayLUB(false); got != 3 {
        t.Fatalf("ArrayLUB(false) = %d, want 3", got)
    }
}
`)
}

func TestConstructorTargetsConditionalLambdaArguments(t *testing.T) {
	src := `
interface IntFn {
    int apply(int value);
}

public class ConstructorConditionalLambdaProgram {
    static class Box {
        final IntFn fn;

        Box(IntFn fn) {
            this.fn = fn;
        }

        int run(int value) {
            return fn.apply(value);
        }
    }

    public static int run(boolean first) {
        Box box = new Box(first ? value -> value + 1 : value -> value + 2);
        return box.run(10);
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "func(value any) any") || strings.Contains(out, "func() any") {
		t.Fatalf("constructor conditional/lambda argument was parsed without its IntFn target:\n%s", out)
	}
	if !strings.Contains(out, "func(value int32) int32") {
		t.Fatalf("constructor lambda parameters/results were not target-typed as int:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestConstructorConditionalLambda(t *testing.T) {
    if got := Run(true); got != 11 {
        t.Fatalf("Run(true) = %d, want 11", got)
    }
    if got := Run(false); got != 12 {
        t.Fatalf("Run(false) = %d, want 12", got)
    }
}
`)
}
