package transpiler

import "testing"

func TestBoxedEvaluationOrderPreservesEarlierReferenceReads(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class BoxedEvaluationOrderProgram {
    static boolean same(Integer left, Integer right) { return left == right; }
    public static boolean identityAfterPostfix() {
        Integer value = 1000;
        return value == value++;
    }
    public static boolean identityAfterPrefix() {
        Integer value = 1000;
        return value != ++value;
    }
    public static boolean equalsAfterPostfix() {
        Integer value = 1000;
        return value.equals(value++);
    }
    public static boolean equalsAfterAssignment() {
        Integer value = 1000;
        return value.equals(value = 1001);
    }
    public static boolean erasedEqualsAfterAssignment() {
        Object value = Integer.valueOf(1000);
        return value.equals(value = Integer.valueOf(1001));
    }
    public static boolean argumentsAfterPostfix() {
        Integer value = 1000;
        return same(value, value++);
    }
    public static int nullReceiverAfterAssignment() {
        Integer value = null;
        try {
            value.equals(value = 1001);
            return 1;
        } catch (NullPointerException expected) {
            return 2;
        }
    }
    public static int erasedNullReceiverAfterAssignment() {
        Object value = null;
        try {
            value.equals(value = Integer.valueOf(1001));
            return 1;
        } catch (NullPointerException expected) {
            return 2;
        }
    }
}
`)
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("generated reference-evaluation program:\n%s", out)
		}
	})
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestEarlierReferenceReads(t *testing.T) {
    for _, test := range []struct {
        name string
        run func() bool
        want bool
    }{
        {"identity after postfix", IdentityAfterPostfix, true},
        {"identity after prefix", IdentityAfterPrefix, true},
        {"equals after postfix", EqualsAfterPostfix, true},
        {"equals after assignment", EqualsAfterAssignment, false},
        {"Object.equals after assignment", ErasedEqualsAfterAssignment, false},
        {"arguments after postfix", ArgumentsAfterPostfix, true},
    } {
        t.Run(test.name, func(t *testing.T) {
            if got := test.run(); got != test.want { t.Fatalf("got %v, want %v", got, test.want) }
        })
    }
    if got := NullReceiverAfterAssignment(); got != 2 { t.Errorf("null wrapper receiver result = %d, want 2", got) }
    if got := ErasedNullReceiverAfterAssignment(); got != 2 { t.Errorf("null Object receiver result = %d, want 2", got) }
}
`)
}

func TestBoxedTernaryRetainsQualifiedWrapperIdentity(t *testing.T) {
	out := renderGoFileFromJava(t, `
class Integer {}
public class BoxedTernaryShadowingProgram {
    public static int nullAndPrimitive(boolean chooseNull) {
        var value = chooseNull ? null : 1;
        return value == null ? 0 : value.intValue();
    }
    public static boolean sourceAndWrapper(boolean chooseSource) {
        Integer source = new Integer();
        java.lang.Integer wrapper = java.lang.Integer.valueOf(1);
        var value = chooseSource ? source : wrapper;
        return chooseSource ? value == source : value == wrapper;
    }
}
`)
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("generated ternary-shadowing program:\n%s", out)
		}
	})
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestTernaryShadowing(t *testing.T) {
    if got := NullAndPrimitive(true); got != 0 { t.Fatalf("null arm = %d, want 0", got) }
    if got := NullAndPrimitive(false); got != 1 { t.Fatalf("primitive arm = %d, want 1", got) }
    if !SourceAndWrapper(true) || !SourceAndWrapper(false) { t.Fatal("reference conditional lost either branch identity") }
}
`)
}

func TestBoxedConstructorMethodReferenceRetainsQualifiedIdentity(t *testing.T) {
	out := renderGoFileFromJava(t, `
class Integer {}
interface IntegerFactory { java.lang.Integer make(int value); }
public class BoxedConstructorReferenceShadowingProgram {
    static java.lang.Integer build() {
        IntegerFactory factory = java.lang.Integer::new;
        return factory.make(123);
    }
    public static boolean run() {
        java.lang.Integer first = build();
        java.lang.Integer second = build();
        return first != second && first.equals(second) && first.intValue() == 123;
    }
}
`)
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("generated qualified-constructor-reference program:\n%s", out)
		}
	})
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestQualifiedConstructorReference(t *testing.T) {
    if !Run() { t.Fatal("qualified Integer constructor reference must create fresh java.lang.Integer objects") }
}
`)
}
