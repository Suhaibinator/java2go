package transpiler

import "testing"

// This is a frozen Java-semantics oracle for the callable side of raw generic
// erasure. Arguments are evaluated left-to-right before the method body, and
// the invocation dispatches virtually to the Derived<X> override. Because X
// and T both erase to Numbered, the raw call may
// store a Second in the same object viewed as Base<First>. Its erased return is
// usable as Numbered; Java inserts the failing First cast only at the later
// exact typed use.
func TestRawGenericCallableABIAdversarial_RawMethodPollutionDispatchAndDelayedCast(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawGenericCallableMethodProbe {
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

    static Numbered argument(int digit, Numbered value) {
        effects = effects * 10 + digit;
        return value;
    }

    static class Base<T extends Numbered> {
        T value;

        Base(T value) {
            this.value = value;
        }

        T exchange(T ignored, T next) {
            effects = effects * 10 + 5;
            T previous = value;
            value = next;
            return previous;
        }

        T current() {
            effects = effects * 10 + 6;
            return value;
        }
    }

    static class Derived<X extends Numbered> extends Base<X> {
        Derived(X value) {
            super(value);
        }

        @Override
        X exchange(X ignored, X next) {
            effects = effects * 10 + 4;
            return super.exchange(ignored, next);
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        effects = 0;
        Base<First> typed = new Derived<First>(new First(1));
        Base raw = typed;

        Numbered previous = (Numbered) raw.exchange(
                argument(2, new First(8)),
                argument(3, new Second(2)));
        Numbered erasedObserved = (Numbered) raw.current();

        String delayedFailure;
        try {
            delayedFailure = "unexpected:" + typed.current().firstOnly();
        } catch (ClassCastException expected) {
            delayedFailure = "ClassCastException";
        }

        return previous.number() + ":" + erasedObserved.number() + ":"
                + effects + ":" + delayedFailure;
    }
}
`, "1:2:234566:ClassCastException")
}

// Raw construction uses the erasures of T for each constructor parameter,
// independently of Go's type inference. Heterogeneous Numbered arguments are
// evaluated left-to-right and initialize one raw Box. The Second remains
// observable through the erased view, while casting the same result to the
// unchecked Box<First> view fails only where First-specific behavior is used.
func TestRawGenericCallableABIAdversarial_RawConstructorErasesParameters(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawGenericCallableConstructorProbe {
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

    static Numbered argument(int digit, Numbered value) {
        effects = effects * 10 + digit;
        return value;
    }

    static class Box<T extends Numbered> {
        T first;
        T last;

        Box(T first, T last) {
            effects = effects * 10 + 3;
            this.first = first;
            this.last = last;
        }

        T readFirst() {
            effects = effects * 10 + 4;
            return first;
        }

        T readLast() {
            effects = effects * 10 + 5;
            return last;
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        effects = 0;
        Box raw = new Box(
                argument(1, new First(1)),
                argument(2, new Second(2)));

        Numbered first = (Numbered) raw.readFirst();
        Numbered last = (Numbered) raw.readLast();
        Box<First> typed = (Box<First>) raw;

        String delayedFailure;
        try {
            delayedFailure = "unexpected:" + typed.readLast().firstOnly();
        } catch (ClassCastException expected) {
            delayedFailure = "ClassCastException";
        }

        return first.number() + ":" + last.number() + ":" + effects + ":"
                + delayedFailure;
    }
}
`, "1:2:123455:ClassCastException")
}

// T... erases to Numbered[], not to one scalar Numbered parameter. A raw
// invocation still creates a non-null array, accepts zero or many arguments,
// evaluates those arguments left-to-right, and exposes normal array length and
// indexing behavior inside the generic method.
func TestRawGenericCallableABIAdversarial_VarargsRetainErasedArraySemantics(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawGenericCallableVarargsProbe {
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

    static Numbered argument(int digit, Numbered value) {
        effects = effects * 10 + digit;
        return value;
    }

    static class Collector<T extends Numbered> {
        int summarize(T... values) {
            effects = effects * 10 + 4;
            if (values == null) {
                return -1;
            }

            int result = values.length * 1000;
            for (int index = 0; index < values.length; index++) {
                Numbered value = values[index];
                result = result + value.number() * (index + 1);
            }
            return result;
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        effects = 0;
        Collector raw = new Collector();
        int populated = raw.summarize(
                argument(1, new First(3)),
                argument(2, new Second(5)),
                argument(3, new First(7)));
        int empty = raw.summarize();
        return populated + ":" + empty + ":" + effects;
    }
}
`, "3034:0:12344")
}
