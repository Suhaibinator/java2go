package transpiler

import (
	"fmt"
	"testing"
)

func assertAnonymousClassAdversarialResult(t *testing.T, source string, want int32) {
	t.Helper()
	out := renderGoFileFromJava(t, source)
	runGeneratedWithStdjava(t, out, fmt.Sprintf(`
package main

import "testing"

func TestAnonymousClassAdversarialResult(t *testing.T) {
	if got := Run(); got != int32(%d) {
		t.Fatalf("Run() = %%d, want %%d", got, int32(%d))
	}
}
`, want, want))
}

// Java evaluates the explicit superclass arguments exactly once from left to
// right, invokes the superclass constructor, and only then runs the anonymous
// class's field initializers. A virtual call made by that constructor already
// dispatches to the anonymous override, but observes the field's default value.
func TestAnonymousClassAdversarial_ConcreteSuperConstructorAndInitializationOrder(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
class AnonymousConstructorOrderBase {
    static int effects;
    int argumentCode;
    int observed;

    AnonymousConstructorOrderBase(int first, int second) {
        effects = effects * 10 + 3;
        argumentCode = first * 10 + second;
        observed = value();
        effects = effects * 10 + 4;
    }

    int value() {
        return -1;
    }
}

public class AnonymousConstructorOrderProgram {
    static int mark(int digit, int value) {
        AnonymousConstructorOrderBase.effects =
            AnonymousConstructorOrderBase.effects * 10 + digit;
        return value;
    }

    public static int run() {
        AnonymousConstructorOrderBase.effects = 0;
        var instance = new AnonymousConstructorOrderBase(mark(1, 4), mark(2, 5)) {
            int field = mark(5, 7);

            int value() {
                AnonymousConstructorOrderBase.effects =
                    AnonymousConstructorOrderBase.effects * 10 + 6;
                return field + 1;
            }
        };

        return AnonymousConstructorOrderBase.effects * 10000
            + instance.argumentCode * 100
            + instance.observed * 10
            + instance.field;
    }
}
`, 1236454517)
}

// An anonymous implementation of a functional interface is only equivalent to
// a closure when it carries no additional object state. A declared field and
// its initializer require a distinct anonymous object whose state is retained
// across calls; this body must therefore bypass the SAM closure fast path.
func TestAnonymousClassAdversarial_StatefulSAMUsesAnonymousObject(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
interface AnonymousStatefulSupplier {
    int get();
}

public class AnonymousStatefulSAMProgram {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    public static int run() {
        effects = 0;
        int seed = 4;
        AnonymousStatefulSupplier supplier = new AnonymousStatefulSupplier() {
            int value = mark(1, seed + 2);

            public int get() {
                effects = effects * 10 + 2;
                return value;
            }
        };

        int first = supplier.get();
        int second = supplier.get();
        return effects * 100 + first * 10 + second;
    }
}
`, 12266)
}

// Anonymous field initializers run in source order. If one completes abruptly,
// later initializers and the post-allocation statement must be skipped while
// the exception propagates to the surrounding catch.
func TestAnonymousClassAdversarial_AbruptFieldInitializerStopsConstruction(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
public class AnonymousAbruptInitializerProgram {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    static int fail(int digit) {
        effects = effects * 10 + digit;
        throw new IllegalStateException("field initializer failed");
    }

    public static int run() {
        effects = 0;
        try {
            var value = new Object() {
                int first = mark(1, 1);
                int second = fail(2);
                int third = mark(3, 3);
            };
            effects = effects * 10 + value.third + 9;
        } catch (IllegalStateException expected) {
            effects = effects * 10 + 4;
        }
        return effects;
    }
}
`, 124)
}
