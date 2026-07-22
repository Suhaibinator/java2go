package transpiler

import (
	"fmt"
	"testing"
)

func assertAnonymousMemberCollisionResult(t *testing.T, source string, want int32) {
	t.Helper()
	out := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, out, fmt.Sprintf(`
package main

import "testing"

func TestAnonymousMemberCollisionResult(t *testing.T) {
	if got := Run(); got != int32(%d) {
		t.Fatalf("Run() = %%d, want %%d", got, int32(%d))
	}
}
`, want, want))
}

// Keep initializer lowering independently pinned so a collision renaming fix
// cannot turn the main fixture green while silently leaving every anonymous
// instance field at its Go zero value. Java runs these initializers in source
// order on each allocation, with both captures and earlier fields in scope.
func TestAnonymousClass_DeclaredFieldInitializersRunInSourceOrder(t *testing.T) {
	assertAnonymousMemberCollisionResult(t, `
public class AnonymousMemberCollisionProgram {
    public static int run() {
        int offset = 3;
        var value = new Object() {
            int first = offset + 1;
            int second = first + 2;

            int read() {
                return first * 10 + second;
            }
        };

        return value.read();
    }
}
`, 46)
}

// Java's exact anonymous type has separate field and method namespaces. A var
// local retains that exact type, so both selectors below are legal and must be
// retargeted if either generated Go member is renamed to avoid a collision.
func TestAnonymousMemberCollision_VarRetainsExactAnonymousType(t *testing.T) {
	assertAnonymousMemberCollisionResult(t, `
public class AnonymousMemberCollisionProgram {
    public static int run() {
        var value = new Object() {
            int score = 4;

            int score() {
                return score;
            }
        };

        return value.score * 10 + value.score();
    }
}
`, 44)
}

// Field initializers execute on every anonymous allocation and can observe
// captured enclosing locals. Calls between sibling methods must still resolve
// through the anonymous receiver after collision-safe member renaming.
func TestAnonymousMemberCollision_FieldInitializerCaptureAndSiblingDispatch(t *testing.T) {
	assertAnonymousMemberCollisionResult(t, `
public class AnonymousMemberCollisionProgram {
    public static int run() {
        int seed = 3;
        var value = new Object() {
            int score = seed + 1;

            int score() {
                return score + seed;
            }

            int dispatch() {
                return score();
            }
        };

        return value.score * 100 + value.dispatch();
    }
}
`, 407)
}

// Anonymous types at different creation sites are distinct even when their
// source members have identical spellings. Their exact-type registrations,
// captures, initializer state, and collision allocations must not bleed into
// one another.
func TestAnonymousMemberCollision_MultipleAnonymousTypesRemainIndependent(t *testing.T) {
	assertAnonymousMemberCollisionResult(t, `
public class AnonymousMemberCollisionProgram {
    public static int run() {
        int base = 2;
        var left = new Object() {
            int value = base + 1;

            int value() {
                return value * 10;
            }
        };
        var right = new Object() {
            int value = base + 4;

            int value() {
                return value * 100;
            }
        };

        return left.value + left.value() + right.value + right.value();
    }
}
`, 639)
}

// Captured locals also become synthetic Go struct fields. Java still permits a
// method with the same spelling because the captured variable and method live
// in different namespaces; the collision allocator must therefore cover
// captures without renaming the enclosing local binding itself.
func TestAnonymousMemberCollision_CapturedLocalAndMethodUseSeparateNamespaces(t *testing.T) {
	assertAnonymousMemberCollisionResult(t, `
public class AnonymousMemberCollisionProgram {
    public static int run() {
        int offset = 5;
        var value = new Object() {
            int offset() {
                return offset;
            }
        };

        return value.offset() * 10 + offset;
    }
}
`, 55)
}

// A collision fix must preserve the method spelling required by a superclass's
// virtual-dispatch contract (or update that contract coherently). Renaming only
// the anonymous method so Go accepts the struct would silently break this call
// through the inherited method.
func TestAnonymousMemberCollision_PreservesSuperclassVirtualDispatch(t *testing.T) {
	assertAnonymousMemberCollisionResult(t, `
abstract class AnonymousCollisionBase {
    abstract int score();

    int dispatch() {
        return score();
    }
}

public class AnonymousMemberCollisionProgram {
    public static int run() {
        int offset = 2;
        var value = new AnonymousCollisionBase() {
            int score = 5 + offset;

            int score() {
                return score + 1;
            }
        };

        return value.score * 10 + value.dispatch();
    }
}
`, 78)
}
