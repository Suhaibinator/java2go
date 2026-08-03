package transpiler

import (
	"strings"
	"testing"
)

// Java constructor delegation performs an ordinary variable-arity invocation.
// Even when this(), explicit super(), or implicit super() supplies no source
// arguments, the target constructor observes a fresh, non-null zero-length
// array rather than Go's nil variadic slice.
func TestConstructorDelegation_EmptyVarargsPreservesJavaArraySemantics(t *testing.T) {
	out := renderGoFileFromJava(t, `
class ThisTarget {
    int size;

    ThisTarget(Object... values) {
        size = values == null ? -1 : values.length;
    }

    ThisTarget(int marker) {
        this();
    }
}

class VarargsBase {
    int size;

    VarargsBase(Object... values) {
        size = values == null ? -1 : values.length;
    }
}

class ExplicitSuperChild extends VarargsBase {
    ExplicitSuperChild(int marker) {
        super();
    }
}

class ImplicitSuperChild extends VarargsBase {
    ImplicitSuperChild(int marker) {
    }
}

class DefaultConstructorChild extends VarargsBase {
}

class GenericVarargsBase {
    int size;

    <E> GenericVarargsBase(E... values) {
        size = values == null ? -1 : values.length;
    }
}

class GenericImplicitSuperChild extends GenericVarargsBase {
    GenericImplicitSuperChild() {
    }
}

public class ConstructorEmptyVarargsProgram {
    public static String run() {
        return new ThisTarget(1).size + ":"
                + new ExplicitSuperChild(2).size + ":"
                + new ImplicitSuperChild(3).size + ":"
                + new DefaultConstructorChild().size + ":"
                + new GenericImplicitSuperChild().size;
    }
}
`)
	if count := strings.Count(out, "stdjava.ArrayLiteral[any]()..."); count < 5 {
		t.Fatalf("empty constructor delegations emitted %d allocated varargs slices, want at least 5:\n%s", count, out)
	}
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestConstructorEmptyVarargsRuntime(t *testing.T) {
    const want = "0:0:0:0:0"
    if got := Run(); got != want {
        t.Fatalf("Run() = %q, want %q", got, want)
    }
}
`)
}
