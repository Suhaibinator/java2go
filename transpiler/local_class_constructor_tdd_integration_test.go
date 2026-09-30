package transpiler

import (
	"fmt"
	"testing"
)

func assertGeneratedLocalConstructorResult(t *testing.T, source, want string) {
	t.Helper()
	out := renderGoFileFromJava(t, source)
	runGeneratedWithStdjava(t, out, fmt.Sprintf(`
package main

import "testing"

func TestLocalConstructorResult(t *testing.T) {
	if got := Run(); got != %q {
		t.Fatalf("Run() = %%q, want %%q", got, %q)
	}
}
`, want, want))
}

func TestLocalClassConstructor_ExplicitArgumentsAndBodySideEffectOrder(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class LocalConstructorProgram {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    public static String run() {
        effects = 0;

        class LocalBox {
            int value;

            LocalBox(int left, int right) {
                effects = effects * 10 + 3;
                value = left * 10 + right;
            }

            int read() {
                return value;
            }
        }

        LocalBox box = new LocalBox(mark(1, 4), mark(2, 5));
        return box.read() + ":" + effects;
    }
}
`, "45:123")
}

func TestLocalClassConstructor_CapturesAndFieldInitializersPrecedeBody(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class LocalConstructorProgram {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    public static String run() {
        int captured = 7;
        effects = 0;

        class LocalBox {
            int first = mark(2, captured);
            int second = mark(3, first + 1);
            int value;

            LocalBox(int input) {
                mark(4, 0);
                value = input + first + second + captured;
            }
        }

        LocalBox box = new LocalBox(mark(1, 5));
        return effects + ":" + box.first + ":" + box.second + ":" + box.value;
    }
}
`, "1234:7:8:27")
}

func TestLocalClassConstructor_OverloadsAndThisDelegationInitializeFieldsOnce(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class LocalConstructorProgram {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    public static String run() {
        effects = 0;

        class LocalValue {
            int seed = mark(5, 1);
            int value;

            LocalValue(int input) {
                this(input, mark(3, 2));
                mark(4, 0);
            }

            LocalValue(int left, int right) {
                mark(2, 0);
                value = left * 10 + right + seed;
            }
        }

        LocalValue result = new LocalValue(mark(1, 4));
        return result.value + ":" + effects;
    }
}
`, "43:13524")
}

func TestLocalClassConstructor_RecursiveAllocationForwardsCaptureAndArguments(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
public class LocalConstructorProgram {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    public static String run() {
        int captured = 3;
        effects = 0;

        class LocalNode {
            int value;

            LocalNode(int depth) {
                mark(depth + 1, 0);
                value = depth == 0 ? captured : new LocalNode(depth - 1).value + captured;
            }
        }

        LocalNode root = new LocalNode(2);
        return root.value + ":" + effects;
    }
}
`, "9:321")
}

func TestLocalClassConstructor_SuperCallPrecedesFieldInitializersAndBody(t *testing.T) {
	assertGeneratedLocalConstructorResult(t, `
class LocalConstructorBase {
    int base;

    LocalConstructorBase(int value) {
        LocalConstructorProgram.mark(3, 0);
        base = value;
    }
}

public class LocalConstructorProgram {
    static int effects;

    static int mark(int digit, int value) {
        effects = effects * 10 + digit;
        return value;
    }

    public static String run() {
        effects = 0;

        class LocalChild extends LocalConstructorBase {
            int local = mark(4, 4);

            LocalChild(int value) {
                super(mark(2, value));
                mark(5, 0);
            }

            int sum() {
                return base + local;
            }
        }

        LocalChild child = new LocalChild(mark(1, 7));
        return child.sum() + ":" + effects;
    }
}
`, "11:12345")
}
