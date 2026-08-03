package transpiler

import "testing"

func TestDirectOwnerTypeParameterFieldErasure_AssignmentExpressionUsesRawStorage(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class DirectFieldAssignmentExpressionProbe {
    interface Numbered {
        int number();
    }

    static class First implements Numbered {
        final int value;

        First(int value) { this.value = value; }

        public int number() { return value; }

        int firstOnly() { return value + 100; }
    }

    static class Second implements Numbered {
        final int value;

        Second(int value) { this.value = value; }

        public int number() { return value; }
    }

    static class Outer<T extends Numbered> {
        T value;

        Outer(T value) { this.value = value; }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Outer<First> typed = new Outer<First>(new First(1));
        First typedResult = (typed.value = new First(2));

        Outer raw = typed;
        Numbered rawResult = (Numbered) (raw.value = new Second(3));

        String delayedFailure;
        try {
            delayedFailure = "unexpected:" + typed.value.firstOnly();
        } catch (ClassCastException expected) {
            delayedFailure = "ClassCastException";
        }
        return typedResult.number() + ":" + rawResult.number() + ":" + delayedFailure;
    }
}
`, "2:3:ClassCastException")
}
