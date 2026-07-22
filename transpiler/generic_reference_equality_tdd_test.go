package transpiler

import (
	"fmt"
	"testing"
)

func assertGenericReferenceEqualityResult(t *testing.T, source string, want int32) {
	t.Helper()
	out := renderGoFileFromJava(t, source)
	runGeneratedWithStdjava(t, out, fmt.Sprintf(`
package main

import "testing"

func TestGenericReferenceEqualityResult(t *testing.T) {
	if got := Run(); got != int32(%d) {
		t.Fatalf("Run() = %%d, want %%d", got, int32(%d))
	}
}
`, want, want))
}

// Boxing a typed nil pointer into a Go interface produces a non-nil interface.
// Java generic reference equality must nevertheless recognize that value as
// null when the source-level T is instantiated with an ordinary class type.
func TestGenericReferenceEquality_TypedNilEqualsNull(t *testing.T) {
	assertGenericReferenceEqualityResult(t, `
class GenericEqualityToken {
}

public class GenericTypedNullProgram {
    static <T> boolean isNull(T value) {
        return value == null;
    }

    public static int run() {
        GenericEqualityToken missing = null;
        return isNull(missing) ? 1 : 0;
    }
}
`, 1)
}

// The inverse comparison must agree with == for both a real object and a
// typed-null reference; merely comparing their boxed Go interface headers is
// not equivalent to Java's reference-null test.
func TestGenericReferenceEquality_TypedNilNotEqualsNull(t *testing.T) {
	assertGenericReferenceEqualityResult(t, `
class GenericInequalityToken {
}

public class GenericTypedNotNullProgram {
    static <T> boolean isPresent(T value) {
        return value != null;
    }

    public static int run() {
        GenericInequalityToken missing = null;
        GenericInequalityToken present = new GenericInequalityToken();
        return (isPresent(missing) ? 90 : 10)
            + (isPresent(present) ? 1 : 0);
    }
}
`, 11)
}

// Built-in Runnable lambdas currently use a function-backed runtime adapter.
// A function is not Go-comparable, but Java still permits reference identity
// comparison after the value flows through a generic T. Comparing a value with
// itself must return true without a Go "comparing uncomparable type" panic.
func TestGenericReferenceEquality_FunctionalAdapterIdentityDoesNotPanic(t *testing.T) {
	assertGenericReferenceEqualityResult(t, `
public class GenericFunctionalIdentityProgram {
    static <T> boolean same(T left, T right) {
        return left == right;
    }

    public static int run() {
        Runnable task = () -> { };
        return same(task, task) ? 1 : 0;
    }
}
`, 1)
}
