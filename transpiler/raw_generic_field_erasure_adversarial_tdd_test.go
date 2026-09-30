package transpiler

import "testing"

// A bare class type-parameter field is stored at its Java erasure. A raw alias
// may therefore pollute the same object with another implementation of the
// first bound. Reading through the raw alias and invoking only the bound from
// inside the generic class must both succeed; the compiler-inserted cast is
// delayed until the exact Outer<First> view needs First-specific behavior.
func TestRawGenericFieldErasureAdversarial_DirectRawWritePollutesTypedAlias(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawGenericFieldErasureProbe {
    interface Numbered {
        int number();
    }

    static class First implements Numbered {
        final int value;

        First(int value) {
            this.value = value;
        }

        public int number() {
            return value;
        }

        int firstOnly() {
            return value + 100;
        }
    }

    static class Second implements Numbered {
        final int value;

        Second(int value) {
            this.value = value;
        }

        public int number() {
            return value;
        }
    }

    static class Outer<T extends Numbered> {
        T value;

        Outer(T value) {
            this.value = value;
        }

        int readThroughFirstBound() {
            return value.number();
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Outer<First> typed = new Outer<First>(new First(1));
        Outer raw = typed;

        raw.value = new Second(2);
        Numbered rawObserved = (Numbered) raw.value;
        int boundOnly = typed.readThroughFirstBound();

        String delayedFailure;
        try {
            delayedFailure = "unexpected:" + typed.value.firstOnly();
        } catch (ClassCastException expected) {
            delayedFailure = "ClassCastException";
        }

        return rawObserved.number() + ":" + boundOnly + ":" + delayedFailure;
    }
}
`, "2:2:ClassCastException")
}

// Java casts null to the concrete type argument without failure. An erased
// field representation must preserve that rule when a default-valued field is
// read through an exact parameterized view.
func TestRawGenericFieldErasureAdversarial_DefaultNullSurvivesTypedRead(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawGenericNullFieldProbe {
    interface Numbered {
        int number();
    }

    static class First implements Numbered {
        public int number() {
            return 1;
        }
    }

    static class Outer<T extends Numbered> {
        T value;

        Outer() {
        }
    }

    public static String run() {
        Outer<First> typed = new Outer<First>();
        First observed = typed.value;
        return observed == null ? "null" : "unexpected";
    }
}
`, "null")
}
