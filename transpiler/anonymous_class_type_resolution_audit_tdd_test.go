package transpiler

import "testing"

// An anonymous class declared inside a generic class and a generic method may
// use both sets of type parameters in its fields and captured state. Hoisting
// that class to file scope must carry the complete parameter set with it.
func TestAnonymousClassTypeAudit_EnclosingAndMethodTypeParameters(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
class GenericAnonymousToken {
}

class GenericAnonymousOuter<T> {
    T outerValue;

    GenericAnonymousOuter(T outerValue) {
        this.outerValue = outerValue;
    }

    <U> int combine(U input) {
        T savedOuter = outerValue;
        U savedInput = input;
        var state = new Object() {
            T left = savedOuter;
            U right = savedInput;

            int code() {
                return (left == savedOuter ? 10 : 0)
                    + (right == savedInput ? 1 : 0);
            }
        };
        return state.code();
    }
}

public class GenericAnonymousProgram {
    public static int run() {
        GenericAnonymousToken left = new GenericAnonymousToken();
        GenericAnonymousToken right = new GenericAnonymousToken();
        return new GenericAnonymousOuter<GenericAnonymousToken>(left).combine(right);
    }
}
`, 11)
}

// Java fields and inherited methods occupy separate namespaces. A field on the
// anonymous class must not mask the promoted Go method inherited from its base.
func TestAnonymousClassTypeAudit_FieldAndInheritedMethodRemainDistinct(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
class InheritedSelectorBase {
    int score() {
        return 7;
    }
}

public class InheritedSelectorProgram {
    public static int run() {
        var value = new InheritedSelectorBase() {
            int score = 4;
        };
        return value.score * 10 + value.score();
    }
}
`, 47)
}

// An embedded Go field is named after its type. Java permits an anonymous field
// with that spelling, so synthetic storage must be renamed without changing the
// Java selector used at the creation site or inside the anonymous body.
func TestAnonymousClassTypeAudit_FieldAvoidsEmbeddedSupertypeName(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
class EmbeddedSelectorBase {
    int read() {
        return 3;
    }
}

public class EmbeddedSelectorProgram {
    public static int run() {
        var value = new EmbeddedSelectorBase() {
            int EmbeddedSelectorBase = 4;

            int sum() {
                return EmbeddedSelectorBase + read();
            }
        };
        return value.EmbeddedSelectorBase * 10 + value.sum();
    }
}
`, 47)
}

// Java identifiers such as type, range, and map are legal field names but Go
// keywords. Synthetic anonymous members need the same identifier sanitation
// and selector retargeting as ordinary class members.
func TestAnonymousClassTypeAudit_JavaFieldsNamedGoKeywords(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
public class AnonymousKeywordFieldProgram {
    public static int run() {
        var value = new Object() {
            int type = 4;
            int range = 5;
            int map = 6;

            int code() {
                return type * 100 + range * 10 + map;
            }
        };
        return value.code();
    }
}
`, 456)
}

// A method declaration name is not a read of a same-spelled enclosing local.
// In particular, the source-only `var` type must never leak into a synthesized
// capture field merely because the anonymous class declares method ping().
func TestAnonymousClassTypeAudit_MethodNameDoesNotCreateCapture(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
public class AnonymousFalseCaptureProgram {
    public static int run() {
        var ping = 9;
        var value = new Object() {
            int ping() {
                return 4;
            }
        };
        return ping * 10 + value.ping();
    }
}
`, 94)
}

// Anonymous superclass construction uses ordinary Java overload phases. Bare
// null selects the more specific String overload, byte widens to int before
// long, and an existing array is expanded into a varargs constructor exactly as
// it is for a non-anonymous creation expression.
func TestAnonymousClassTypeAudit_SuperConstructorOverloadsAndVarargs(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
class ReferenceOverloadBase {
    int selected;

    ReferenceOverloadBase(Object value) {
        selected = 1;
    }

    ReferenceOverloadBase(String value) {
        selected = 2;
    }
}

class NumericOverloadBase {
    int selected;

    NumericOverloadBase(long value) {
        selected = 3;
    }

    NumericOverloadBase(int value) {
        selected = 4;
    }
}

class VarargsOverloadBase {
    int selected;

    VarargsOverloadBase(int... values) {
        selected = values.length * 10 + values[0];
    }
}

public class AnonymousConstructorOverloadProgram {
    public static int run() {
        var reference = new ReferenceOverloadBase(null) {
        };
        var numeric = new NumericOverloadBase((byte) 7) {
        };
        int[] packed = new int[] { 6, 7 };
        var variadic = new VarargsOverloadBase(packed) {
        };
        return reference.selected * 1000
            + numeric.selected * 100
            + variadic.selected;
    }
}
`, 2426)
}

// The type of an anonymous creation expression is its exact anonymous type even
// without an intervening var local. Registration must therefore happen before
// resolving an immediate field or method selector, including collision-renamed
// fields at two independent creation sites.
func TestAnonymousClassTypeAudit_ImmediateExactFieldAndMethodAccess(t *testing.T) {
	assertAnonymousClassAdversarialResult(t, `
public class AnonymousImmediateSelectorProgram {
    public static int run() {
        int field = (new Object() {
            int value = 4;

            int value() {
                return 9;
            }
        }).value;
        int method = (new Object() {
            int value = 8;

            int value() {
                return 5;
            }
        }).value();
        return field * 10 + method;
    }
}
`, 45)
}
