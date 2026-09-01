package transpiler

import "testing"

// A raw view is an alias of the same Java object, not a separately
// instantiated generic object. The raw member invocation therefore accepts
// the erasure of T (Numbered), can store a different Numbered implementation,
// and exposes that value through the raw alias. The heap pollution is detected
// only when a later read through the original Outer<First> view performs
// Java's compiler-inserted cast back to First.
func TestRawGenericAliasAdversarial_MemberWritePollutesExactTypedAlias(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RawGenericAliasProbe {
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

        class Inner {
            Inner() {
            }

            T replace(T next) {
                T previous = value;
                value = next;
                return previous;
            }

            T observe() {
                return value;
            }
        }
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static String run() {
        Outer<First> typed = new Outer<First>(new First(1));
        Outer raw = typed;
        Outer.Inner rawInner = raw.new Inner();

        Numbered previous = (Numbered) rawInner.replace(new Second(2));
        Numbered rawObserved = (Numbered) rawInner.observe();

        String delayedFailure;
        try {
            delayedFailure = "unexpected:" + typed.value.firstOnly();
        } catch (ClassCastException expected) {
            delayedFailure = "ClassCastException";
        }

        return previous.number() + ":" + rawObserved.number() + ":" + delayedFailure;
    }
}
`, "1:2:ClassCastException")
}
