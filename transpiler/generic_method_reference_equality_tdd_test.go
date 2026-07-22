package transpiler

import "testing"

// A target-typed Runnable method reference is one Java object. Passing the same
// local through two generic arguments must preserve that allocation identity,
// while the adapter must still forward the caller's execution token when run.
func TestGenericReferenceEquality_MethodReferenceAdapterIdentity(t *testing.T) {
	assertGenericReferenceEqualityResult(t, `
public class GenericMethodReferenceIdentityProgram {
    int calls;

    void noop() {
        calls++;
    }

    static <T> boolean same(T left, T right) {
        return left == right;
    }

    int evaluate() {
        Runnable task = this::noop;
        boolean identical = same(task, task);
        task.run();
        return (identical ? 10 : 90) + calls;
    }

    public static int run() {
        return new GenericMethodReferenceIdentityProgram().evaluate();
    }
}
`, 11)
}
