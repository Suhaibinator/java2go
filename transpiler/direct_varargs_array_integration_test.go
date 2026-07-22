package transpiler

import (
	"strings"
	"testing"
)

// This program is intentionally broad: Java treats a single array argument to
// a varargs declaration as a fixed-arity invocation, while Go requires the
// generated wrapper backing slice to be expanded explicitly. Ordinary element
// arguments must continue to use Go's normal variadic calling convention.
func TestDirectVarargsArrayCalls_PreserveJavaInvocationSemantics(t *testing.T) {
	src := `
public class DirectVarargsArrayProgram {
    static int trace;

    static class Worker {
        int primitive(int... values) {
            int total = 0;
            for (int value : values) total += value;
            return total;
        }

        int reference(String... values) {
            int total = 0;
            for (String value : values) total = total * 10 + value.length();
            return total;
        }
    }

    static class PrimitiveBox {
        int value;
        PrimitiveBox(int... values) {
            for (int item : values) value += item;
        }
    }

    static class ReferenceBox {
        int value;
        ReferenceBox(String... values) {
            for (String item : values) value = value * 10 + item.length();
        }
    }

    static Worker worker(int digit) {
        trace = trace * 10 + digit;
        return new Worker();
    }

    static int[] primitiveArray(int digit) {
        trace = trace * 10 + digit;
        return new int[] { digit, digit + 1 };
    }

    static String[] referenceArray(int digit) {
        trace = trace * 10 + digit;
        return new String[] { "a", "bbb" };
    }

    static int primitive(int... values) {
        int total = 0;
        for (int value : values) total += value;
        return total;
    }

    static int reference(String... values) {
        int total = 0;
        for (String value : values) total = total * 10 + value.length();
        return total;
    }

    public static String run() {
        trace = 0;
        int[] numbers = new int[] { 2, 3, 5 };
        String[] words = new String[] { "aa", "b", "cccc" };
        Worker direct = new Worker();

        int unqualifiedPrimitive = primitive(numbers);
        int qualifiedPrimitive = DirectVarargsArrayProgram.primitive(numbers);
        int instancePrimitive = direct.primitive(numbers);
        int primitiveConstructor = new PrimitiveBox(numbers).value;

        int unqualifiedReference = reference(words);
        int qualifiedReference = DirectVarargsArrayProgram.reference(words);
        int instanceReference = direct.reference(words);
        int referenceConstructor = new ReferenceBox(words).value;

        int ordinaryPrimitive = primitive(7, 8, 9);
        int ordinaryReference = reference("x", "yy", "zzz");

        int orderedPrimitive = worker(1).primitive(primitiveArray(2));
        int orderedReference = worker(3).reference(referenceArray(4));

        return trace + "|" +
            unqualifiedPrimitive + "," + qualifiedPrimitive + "," +
            instancePrimitive + "," + primitiveConstructor + "|" +
            unqualifiedReference + "," + qualifiedReference + "," +
            instanceReference + "," + referenceConstructor + "|" +
            ordinaryPrimitive + "," + ordinaryReference + "|" +
            orderedPrimitive + "," + orderedReference;
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "stdjava.PrimitiveArrayElements(numbers)...") {
		t.Fatalf("direct primitive-array varargs call was not unwrapped and expanded:\n%s", out)
	}
	if !strings.Contains(flat, "stdjava.ReferenceArrayElements[string](words, stdjava.StringTypeID)...") {
		t.Fatalf("direct reference-array varargs call was not converted and expanded:\n%s", out)
	}
	if strings.Contains(flat, "stdjava.PrimitiveArrayElements(int32(7))") ||
		strings.Contains(flat, "stdjava.ReferenceArrayElements[string](\"x\"") {
		t.Fatalf("ordinary element varargs arguments were incorrectly treated as arrays:\n%s", out)
	}

	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestDirectVarargsArrayParity(t *testing.T) {
    const want = "1234|10,10,10,10|214,214,214,214|24,123|5,13"
    if got := Run(); got != want {
        t.Fatalf("Run() = %q, want %q", got, want)
    }
}
`)
}

func TestDirectVarargsArrayCalls_RespectFixedArityOverloadPhase(t *testing.T) {
	src := `
public class DirectVarargsOverloadProgram {
    static int pick(Object value) { return 1; }
    static int pick(String... values) { return 2; }

    static int objectPick(Object... values) { return 4; }

    public static int run() {
        int result = 0;
        result = result * 10 + pick("scalar");
        result = result * 10 + pick(new String[] { "array" });
        result = result * 10 + objectPick(new String[] { "covariant" });
        return result;
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestDirectVarargsOverloadParity(t *testing.T) {
    if got := Run(); got != 124 {
        t.Fatalf("Run() = %d, want 124", got)
    }
}
`)
}

func TestDirectVarargsObjectOverloadTDD_DistinguishesScalarAndArraySignatures(t *testing.T) {
	src := `
public class DirectVarargsObjectOverloadProgram {
    static int pick(Object value) { return 3; }
    static int pick(Object... values) { return 4; }

    public static int run() {
        int result = 0;
        result = result * 10 + pick("scalar");
        result = result * 10 + pick(new Object[] { "array" });
        result = result * 10 + pick(new String[] { "covariant" });
        return result;
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestDirectVarargsObjectOverloadParity(t *testing.T) {
    if got := Run(); got != 344 {
        t.Fatalf("Run() = %d, want 344", got)
    }
}
`)
}
