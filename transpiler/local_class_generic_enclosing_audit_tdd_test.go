package transpiler

import "testing"

// Even though LocalValue never mentions T directly, its enclosing-instance
// link has type GenericOuterLocalLinkProgram<T>. Hoisting the local class must
// therefore carry T so the link remains well-typed and points at this object.
func TestLocalClassAudit_GenericOuterLinkCarriesOtherwiseUnusedTypeParameter(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class GenericOuterLocalLinkProgram<T> {
    int base;

    GenericOuterLocalLinkProgram(int base) {
        this.base = base;
    }

    int compute() {
        class LocalValue {
            int read() {
                return GenericOuterLocalLinkProgram.this.base;
            }
        }

        return new LocalValue().read();
    }

    public static String run() {
        return new GenericOuterLocalLinkProgram<String>(7).compute()
            + ":"
            + new GenericOuterLocalLinkProgram<Integer>(4).compute();
    }
}
`, "7:4")
}

// A recursive allocation inside a local-class instance method is still
// lexically enclosed by the original top-level object. Its synthetic outer
// argument must come from the receiver's stored outer link, not from the local
// receiver itself.
func TestLocalClassAudit_RecursiveAllocationForwardsStoredOuterInstance(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class RecursiveOuterLinkProgram {
    int base;

    RecursiveOuterLinkProgram(int base) {
        this.base = base;
    }

    int compute(int depth) {
        class LocalValue {
            int sum(int remaining) {
                if (remaining == 0) {
                    return RecursiveOuterLinkProgram.this.base;
                }
                return RecursiveOuterLinkProgram.this.base
                    + new LocalValue().sum(remaining - 1);
            }
        }

        return new LocalValue().sum(depth);
    }

    public static String run() {
        return new RecursiveOuterLinkProgram(5).compute(2)
            + ":"
            + new RecursiveOuterLinkProgram(7).compute(1);
    }
}
`, "15:14")
}

// Method type parameters used by a local class's superclass are part of the
// hoisted class's signature even when Java infers them at construction sites.
// T's bound is itself the method parameter B, whose Root bound must remain
// usable transitively by Holder<T> and by calls through T.
func TestLocalClassAudit_HeaderCarriesTransitivelyBoundMethodTypeParameter(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class LocalHeaderTypeParameterProgram {
    interface Root {
        int value();
    }

    static class Impl implements Root {
        int number;

        Impl(int number) {
            this.number = number;
        }

        public int value() {
            return number;
        }
    }

    static class Holder<U extends Root> {
        U held;

        Holder(U held) {
            this.held = held;
        }
    }

    static <B extends Root, T extends B> int score(B baseView, T concrete) {
        class LocalValue extends Holder<T> {
            LocalValue() {
                super(concrete);
            }

            int combined() {
                return baseView.value() * 10 + held.value();
            }
        }

        return new LocalValue().combined();
    }

    public static String run() {
        return score(new Impl(3), new Impl(7))
            + ":"
            + score(new Impl(2), new Impl(9));
    }
}
`, "37:29")
}
