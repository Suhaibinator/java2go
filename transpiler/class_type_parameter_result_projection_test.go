package transpiler

import "testing"

func TestDirectOwnerTypeParameterResult_NarrowsOnlyForConcreteMemberUse(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class GenericResultProjectionProgram {
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

    static int effects;

    static class Box<T extends Numbered> {
        final T value;

        Box(T value) {
            this.value = value;
        }

        T read() {
            effects++;
            return value;
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        effects = 0;
        Box raw = new Box(new Second(4));
        Box<First> typed = (Box<First>) raw;

        Numbered broadValue = typed.read();
        int broad = broadValue.number();

        String boundFailure;
        try {
            boundFailure = "unexpected:" + typed.read().number();
        } catch (ClassCastException expected) {
            boundFailure = "ClassCastException";
        }

        String concreteFailure;
        try {
            concreteFailure = "unexpected:" + typed.read().firstOnly();
        } catch (ClassCastException expected) {
            concreteFailure = "ClassCastException";
        }

        return broad + ":" + effects + ":" + boundFailure + ":" + concreteFailure;
    }
}
`, "4:3:ClassCastException:ClassCastException")
}
