package transpiler

import (
	"strings"
	"testing"
)

func TestFinallyLoopControl_RuntimeParity(t *testing.T) {
	src := `
class AbruptCloseResource implements AutoCloseable {
    public void close() {
        throw new IllegalStateException();
    }
}

public class FinallyLoopControlProgram {
    public static int basicOuterTransfers() {
        int trace = 0;
        for (int index = 0; index < 3; index++) {
            try {
                trace = trace * 10 + index + 1;
                if (index == 0) {
                    continue;
                }
                if (index == 1) {
                    break;
                }
            } finally {
                trace = trace * 10 + index + 4;
            }
        }
        return trace;
    }

    public static int localLoopAndSwitchTargets() {
        int trace = 0;
        try {
            for (int index = 0; index < 4; index++) {
                switch (index) {
                    case 0:
                        trace = trace + 1;
                        break;
                    case 1:
                        continue;
                    case 2:
                        trace = trace + 10;
                        break;
                    default:
                        trace = trace + 100;
                        break;
                }
                if (index == 2) {
                    break;
                }
                trace = trace + 1000;
            }
            trace = trace + 10000;
        } finally {
            trace = trace + 100000;
        }
        return trace;
    }

    public static int labeledOuterTransfers() {
        int trace = 0;
        outer:
        for (int row = 0; row < 3; row++) {
            for (int column = 0; column < 3; column++) {
                try {
                    trace = trace * 10 + row + 1;
                    if (row == 0) {
                        continue outer;
                    }
                    break outer;
                } finally {
                    trace = trace * 10 + column + 7;
                }
            }
        }
        return trace;
    }

    public static int breakTargetsOuterSwitch() {
        int trace = 0;
        switch (1) {
            case 1:
                try {
                    trace = 1;
                    break;
                } finally {
                    trace = trace * 10 + 2;
                }
            default:
                trace = 99;
        }
        return trace;
    }

    public static int finallyReturnOverridesBreak() {
        for (int index = 0; index < 1; index++) {
            try {
                break;
            } finally {
                return 41;
            }
        }
        return -1;
    }

    public static int finallyThrowOverridesContinue() {
        try {
            for (int index = 0; index < 1; index++) {
                try {
                    continue;
                } finally {
                    throw new IllegalStateException();
                }
            }
        } catch (IllegalStateException ex) {
            return 42;
        }
        return -1;
    }

    public static int finallyContinueOverridesReturn() {
        int trace = 0;
        for (int index = 0; index < 2; index++) {
            try {
                return -1;
            } finally {
                trace = trace * 10 + index + 1;
                continue;
            }
        }
        return trace;
    }

    public static int finallyBreakOverridesThrow() {
        int trace = 0;
        for (int index = 0; index < 1; index++) {
            try {
                throw new IllegalStateException();
            } finally {
                trace = 43;
                break;
            }
        }
        return trace;
    }

    public static int catchBreakRunsFinally() {
        int trace = 0;
        for (int index = 0; index < 1; index++) {
            try {
                trace = 1;
                throw new IllegalStateException();
            } catch (IllegalStateException ex) {
                trace = trace * 10 + 2;
                break;
            } finally {
                trace = trace * 10 + 3;
            }
        }
        return trace;
    }

    public static int finallyBreakOverridesCatchThrow() {
        int trace = 0;
        for (int index = 0; index < 1; index++) {
            try {
                throw new IllegalArgumentException();
            } catch (IllegalArgumentException ex) {
                throw new IllegalStateException();
            } finally {
                trace = 44;
                break;
            }
        }
        return trace;
    }

    public static int finallyReturnOverridesCatchThrow() {
        try {
            throw new IllegalArgumentException();
        } catch (IllegalArgumentException ex) {
            throw new IllegalStateException();
        } finally {
            return 45;
        }
    }

    public static int resourceCloseThrowOverridesBreak() {
        try {
            for (int index = 0; index < 1; index++) {
                try (AbruptCloseResource resource = new AbruptCloseResource()) {
                    break;
                }
            }
        } catch (IllegalStateException ex) {
            return 46;
        }
        return -1;
    }

    public static int resourceCloseThrowOverridesReturn() {
        try {
            try (AbruptCloseResource resource = new AbruptCloseResource()) {
                return -1;
            }
        } catch (IllegalStateException ex) {
            return 47;
        }
    }

    public static int nestedTryTargetsLocalLoop() {
        int trace = 0;
        try {
            for (int index = 0; index < 2; index++) {
                try {
                    trace = trace * 10 + 1;
                    continue;
                } finally {
                    trace = trace * 10 + 2;
                }
            }
        } finally {
            trace = trace * 10 + 3;
        }
        return trace;
    }

    public static int nestedFinallyContinue() {
        int trace = 0;
        for (int index = 0; index < 2; index++) {
            try {
                try {
                    trace = trace * 10 + 1;
                    continue;
                } finally {
                    trace = trace * 10 + 2;
                }
            } finally {
                trace = trace * 10 + 3;
            }
        }
        return trace;
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "UNSUPPORTED") {
		t.Fatalf("expected finally loop-control lowering without placeholders, got:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestFinallyLoopControlParity(t *testing.T) {
	tests := []struct {
		name string
		got  int32
		want int32
	}{
		{"basic outer transfers", BasicOuterTransfers(), 1425},
		{"local loop and switch targets", LocalLoopAndSwitchTargets(), 111011},
		{"labeled outer transfers", LabeledOuterTransfers(), 1727},
		{"break targets outer switch", BreakTargetsOuterSwitch(), 12},
		{"finally return overrides break", FinallyReturnOverridesBreak(), 41},
		{"finally throw overrides continue", FinallyThrowOverridesContinue(), 42},
		{"finally continue overrides return", FinallyContinueOverridesReturn(), 12},
		{"finally break overrides throw", FinallyBreakOverridesThrow(), 43},
		{"catch break runs finally", CatchBreakRunsFinally(), 123},
		{"finally break overrides catch throw", FinallyBreakOverridesCatchThrow(), 44},
		{"finally return overrides catch throw", FinallyReturnOverridesCatchThrow(), 45},
		{"resource close throw overrides break", ResourceCloseThrowOverridesBreak(), 46},
		{"resource close throw overrides return", ResourceCloseThrowOverridesReturn(), 47},
		{"nested try targets local loop", NestedTryTargetsLocalLoop(), 12123},
		{"nested finally continue", NestedFinallyContinue(), 123123},
	}
	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("%s: got %d, want %d", test.name, test.got, test.want)
		}
	}
}
`)
}

func TestFinallyLoopControl_DoesNotDeclareUnusedControlForOrdinaryTry(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class OrdinaryTryProgram {
    public static int run() {
        try {
            return 7;
        } finally {
            int local = 1;
        }
    }
}
`)
	if strings.Contains(out, "__java2goControl") {
		t.Fatalf("ordinary try unexpectedly declared a loop-control channel:\n%s", out)
	}
}
