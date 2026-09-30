package transpiler

import "testing"

// Constructor selection for a method-local class must use the same Java
// invocation phases as an ordinary constructor. Keep the unrelated null,
// primitive-widening, and variable-arity cases in separate local classes so
// each result identifies the phase that regressed.
func TestLocalClassAdversarial_ConstructorOverloadPhases(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class LocalConstructorAdversarialProgram {
    public static String run() {
        class NullChoice {
            int code;

            NullChoice(Object ignored) { code = 1; }
            NullChoice(String ignored) { code = 2; }
        }

        class NumericChoice {
            int code;

            NumericChoice(long value) { code = 10 + (int) value; }
            NumericChoice(double value) { code = 99; }
        }

        class VarargChoice {
            int code;

            VarargChoice(int value) { code = 20 + value; }
            VarargChoice(int... values) { code = 30 + values.length; }
        }

        NullChoice nullChoice = new NullChoice(null);
        NumericChoice numericChoice = new NumericChoice(4);
        VarargChoice fixedChoice = new VarargChoice(6);
        VarargChoice expandedChoice = new VarargChoice(1, 2, 3);
        return nullChoice.code + ":" + numericChoice.code + ":"
            + fixedChoice.code + ":" + expandedChoice.code;
    }
}
`, "2:14:26:33")
}

// Java evaluates constructor arguments exactly once, from left to right. An
// abrupt argument prevents every later argument, the allocation, field phases,
// and the constructor body from running.
func TestLocalClassAdversarial_ConstructorArgumentsAreSingleEvaluationAndAbrupt(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class LocalConstructorAdversarialProgram {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    static int fail(int digit) {
        effects = effects * 10 + digit;
        throw new IllegalStateException("boom");
    }

    public static String run() {
        class LocalValue {
            int sum;

            LocalValue(int first, int second, int third) {
                effects = effects * 10 + 9;
                sum = first + second + third;
            }
        }

        effects = 0;
        int abruptEffects = -1;
        try {
            new LocalValue(mark(1, 1), fail(2), mark(3, 3));
        } catch (IllegalStateException expected) {
            abruptEffects = effects;
        }

        effects = 0;
        LocalValue completed = new LocalValue(
            mark(4, 4), mark(5, 5), mark(6, 6));
        return abruptEffects + ":" + effects + ":" + completed.sum;
    }
}
`, "12:4569:15")
}

// A constructor or method parameter shadows an enclosing local only inside
// that parameter's scope. The field initializer still captures the enclosing
// seed. Lowering therefore needs distinct, collision-safe storage for the
// hidden capture and the explicit parameters that share its Java name.
func TestLocalClassAdversarial_CaptureSurvivesConstructorAndMethodShadowing(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class LocalConstructorAdversarialProgram {
    public static String run() {
        int seed = 7;

        class LocalValue {
            int fromOuter = seed;
            int fromArgument;

            LocalValue(int seed) {
                fromArgument = seed;
            }

            int encoded(int seed) {
                return fromOuter * 100 + fromArgument * 10 + seed;
            }
        }

        LocalValue value = new LocalValue(5);
        return value.encoded(3) + ":" + seed;
    }
}
`, "753:7")
}

// With no declared constructor, Java synthesizes one that calls super() before
// the local class's field initializers. Both phases run independently for every
// allocation, and the captured value is forwarded to each new instance.
func TestLocalClassAdversarial_ImplicitConstructorRunsSuperAndFieldsPerAllocation(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
class LocalImplicitBase {
    int base;

    LocalImplicitBase() {
        LocalConstructorAdversarialProgram.mark(1, 0);
        base = 5;
    }
}

public class LocalConstructorAdversarialProgram {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    public static String run() {
        int captured = 7;
        effects = 0;

        class LocalValue extends LocalImplicitBase {
            int local = mark(2, captured);

            int sum() {
                return base + local;
            }
        }

        LocalValue first = new LocalValue();
        LocalValue second = new LocalValue();
        return first.sum() + ":" + second.sum() + ":" + effects;
    }
}
`, "12:12:1212")
}

// Hoisting a local class out of a generic method must also hoist or specialize
// the enclosing method type parameter; leaving a file-scope field typed as an
// unbound T is invalid Go.
func TestLocalClassAdversarial_GenericMethodTypeParameterSurvivesHoisting(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class LocalConstructorAdversarialProgram {
    static <T> T roundTrip(T input) {
        class LocalValue {
            T stored;

            LocalValue(T stored) {
                this.stored = stored;
            }

            T get() {
                return stored;
            }
        }

        return new LocalValue(input).get();
    }

    public static String run() {
        return roundTrip("generic") + ":" + roundTrip(7);
    }
}
`, "generic:7")
}
