package transpiler

import "testing"

// A captured Java local can legally use the same spelling as a synthetic Go
// identity hook. The capture must remain readable while the local class still
// carries its exact component identity for reference-array store checks.
func TestLocalClassAdversarial_CapturedIdentityHookNameRetainsNominalIdentity(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class LocalClassAdversarialFollowupProgram {
    static class Wrong { }

    public static String run() {
        int JavaDynamicTypeID = 7;

        class LocalValue {
            int initialized = JavaDynamicTypeID;

            int encoded() {
                return JavaDynamicTypeID * 10 + initialized;
            }
        }

        LocalValue[] exact = new LocalValue[] { new LocalValue() };
        Object[] erased = exact;
        int score = exact[0].encoded();
        try {
            erased[0] = new Wrong();
            score += 1000;
        } catch (ArrayStoreException expected) {
            score += 100;
        }
        erased[0] = new LocalValue();
        return score + ":" + exact[0].encoded();
    }
}
`, "177:77")
}

// Hoisting a class from a generic method must preserve T through field
// initialization, a local-class method signature, and recursive allocations
// that forward the enclosing generic value.
func TestLocalClassAdversarial_GenericRecursiveAllocationPreservesTypeParameter(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class LocalClassAdversarialFollowupProgram {
    static <T> T recursiveLast(T input, int depth) {
        class LocalNode {
            T stored = input;
            LocalNode next;

            LocalNode(int remaining) {
                next = remaining == 0 ? null : new LocalNode(remaining - 1);
            }

            T last() {
                return next == null ? stored : next.last();
            }
        }

        return new LocalNode(depth).last();
    }

    public static String run() {
        return recursiveLast("recursive", 3) + ":" + recursiveLast(9, 2);
    }
}
`, "recursive:9")
}

// A local class declared in another local class's instance method has two
// distinct enclosing instances: its immediately enclosing local object and
// the lexically enclosing top-level object. Both links, plus the method-local
// capture, must survive hoisting.
func TestLocalClassAdversarial_NestedLocalRetainsEnclosingInstances(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class LocalClassAdversarialFollowupProgram {
    int base;

    LocalClassAdversarialFollowupProgram(int base) {
        this.base = base;
    }

    String compute(int seed) {
        class OuterLocal {
            int delta = 2;

            int evaluate() {
                class InnerLocal {
                    int encoded() {
                        return base * 100
                            + LocalClassAdversarialFollowupProgram.this.base * 10
                            + delta + seed;
                    }
                }

                return new InnerLocal().encoded();
            }
        }

        return String.valueOf(new OuterLocal().evaluate());
    }

    public static String run() {
        return new LocalClassAdversarialFollowupProgram(3).compute(4);
    }
}
`, "336")
}
