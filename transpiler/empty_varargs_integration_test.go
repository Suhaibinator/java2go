package transpiler

import (
	"strings"
	"testing"
)

// A Java variable-arity invocation always creates a non-null array, even when
// it supplies no elements. The allocation is fresh per invocation. Supplying a
// null or existing array as the final argument is instead fixed-arity; this test
// pins the existing nullness and length behavior for those calls as well.
func TestEmptyVarargsInvocation_PreservesArraySemanticsAcrossCallABIs(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class EmptyVarargsProgram {
    static int primitive(int... values) {
        return values == null ? -1 : values.length;
    }

    static int prefixed(int prefix, Object... values) {
        return values == null ? -1 : prefix * 10 + values.length;
    }

    static <E> int generic(E... values) {
        return values == null ? -1 : values.length;
    }

    static class Box<T> {
        int size(T... values) {
            return values == null ? -1 : values.length;
        }
    }

    static class Holder<T> {
        final int size;

        Holder(T... values) {
            size = values == null ? -1 : values.length;
        }
    }

    public static String run() {
        Object[] missing = null;
        Object[] explicitEmpty = new Object[0];
        Box<String> box = new Box<>();

        return primitive() + ":" + prefixed(7) + ":"
                + primitive((int[]) null) + ":"
                + prefixed(8, missing) + ":"
                + prefixed(9, explicitEmpty) + ":"
                + generic() + ":"
                + EmptyVarargsProgram.<String>generic() + ":"
                + box.size() + ":" + new Holder<String>().size;
    }
}
`)
	// ArrayLiteral's retained capacity gives each emitted zero-element call its
	// own backing identity; its stdjava unit test pins that runtime property.
	if count := strings.Count(out, "stdjava.ArrayLiteral["); count < 6 {
		t.Fatalf("zero-element varargs calls emitted %d allocated slices, want at least 6:\n%s", count, out)
	}
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestEmptyVarargsBehavior(t *testing.T) {
    const want = "0:70:-1:-1:90:0:0:0:0"
    if got := Run(); got != want {
        t.Fatalf("Run() = %q, want %q", got, want)
    }
}
`)
}
