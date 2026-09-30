package transpiler

import (
	"strings"
	"testing"
)

func TestDoWhileContinue_RuntimeParity(t *testing.T) {
	src := `
public class DoWhileContinueProgram {
    private static int conditionCalls;

    private static boolean condition(int iteration, int limit) {
        conditionCalls++;
        return iteration < limit;
    }

    public static int ordinaryContinueChecksCondition() {
        conditionCalls = 0;
        int iterations = 0;
        do {
            iterations++;
            if (iterations < 3) {
                continue;
            }
        } while (condition(iterations, 3));
        return iterations * 100 + conditionCalls;
    }

    public static int labeledContinueOnly() {
        conditionCalls = 0;
        int iterations = 0;
        outer:
        do {
            iterations++;
            for (int inner = 0; inner < 2; inner++) {
                if (inner == 0) {
                    continue outer;
                }
            }
        } while (condition(iterations, 2));
        return iterations * 100 + conditionCalls;
    }

    public static int labeledContinueAndBreak() {
        conditionCalls = 0;
        int iterations = 0;
        outer:
        do {
            iterations++;
            if (iterations == 1) {
                continue outer;
            }
            break outer;
        } while (condition(iterations, 3));
        return iterations * 100 + conditionCalls;
    }

    public static int continueThroughFinally() {
        conditionCalls = 0;
        int iterations = 0;
        int trace = 0;
        do {
            iterations++;
            try {
                trace = trace * 10 + iterations;
                if (iterations < 3) {
                    continue;
                }
            } finally {
                trace = trace * 10 + 9;
            }
        } while (condition(iterations, 3));
        return trace + conditionCalls;
    }

    public static int labeledContinueThroughFinally() {
        conditionCalls = 0;
        int iterations = 0;
        int trace = 0;
        outer:
        do {
            iterations++;
            try {
                trace = trace * 10 + iterations;
                continue outer;
            } finally {
                trace = trace * 10 + 8;
            }
        } while (condition(iterations, 2));
        return trace + conditionCalls;
    }

    public static int nestedLoopContinueStaysLocal() {
        conditionCalls = 0;
        int iterations = 0;
        int innerWork = 0;
        do {
            iterations++;
            for (int inner = 0; inner < 3; inner++) {
                if (inner < 2) {
                    continue;
                }
                innerWork++;
            }
        } while (condition(iterations, 2));
        return iterations * 100 + innerWork * 10 + conditionCalls;
    }

    public static int breakSkipsCondition() {
        conditionCalls = 0;
        do {
            break;
        } while (condition(0, 1));
        return conditionCalls;
    }

    public static int labeledBreakSkipsCondition() {
        conditionCalls = 0;
        outer:
        do {
            while (true) {
                break outer;
            }
        } while (condition(0, 1));
        return conditionCalls;
    }

    public static int returnSkipsCondition() {
        conditionCalls = 0;
        do {
            return 700 + conditionCalls;
        } while (condition(0, 1));
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "UNSUPPORTED") {
		t.Fatalf("expected do-while control lowering without placeholders, got:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestDoWhileContinueParity(t *testing.T) {
	tests := []struct {
		name string
		got  int32
		want int32
	}{
		{"ordinary continue checks condition", OrdinaryContinueChecksCondition(), 303},
		{"labeled continue only", LabeledContinueOnly(), 202},
		{"labeled continue and break", LabeledContinueAndBreak(), 201},
		{"continue through finally", ContinueThroughFinally(), 192942},
		{"labeled continue through finally", LabeledContinueThroughFinally(), 1830},
		{"nested loop continue stays local", NestedLoopContinueStaysLocal(), 222},
		{"break skips condition", BreakSkipsCondition(), 0},
		{"labeled break skips condition", LabeledBreakSkipsCondition(), 0},
		{"return skips condition", ReturnSkipsCondition(), 700},
	}
	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("%s: got %d, want %d", test.name, test.got, test.want)
		}
	}
}
`)
}

func TestDoWhileContinue_AffineVersionedBodyUsesRenderUniqueLabels(t *testing.T) {
	src := `
final class DoAffineGrid {
    private final double[] values;
    private final int size = 1;

    DoAffineGrid(double value) {
        this.values = new double[1];
        this.values[0] = value;
    }

    double get(int row, int column) {
        return this.values[row * this.size + column];
    }
}

public class DoWhileAffineVersionProgram {
    private static int conditionCalls;

    private static boolean condition(int iteration) {
        conditionCalls++;
        return iteration < 2;
    }

    public static DoAffineGrid allocated(double value) {
        return new DoAffineGrid(value);
    }

    public static double run(DoAffineGrid grid, boolean access) {
        conditionCalls = 0;
        double total = 0.0;
        for (int once = 0; once < 1; once++) {
            if (access) {
                total += grid.get(0, 0);
            }
            int iterations = 0;
            do {
                iterations++;
                if (iterations < 2) {
                    continue;
                }
                total += iterations;
            } while (condition(iterations));
            total += conditionCalls * 10;
        }
        return total;
    }
}
`

	out := renderGoFileFromJava(t, src)
	if !strings.Contains(normalizeSpaces(out), "} else {") {
		t.Fatalf("outer accessor loop was not affine-versioned:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestAffineVersionedDoWhileParity(t *testing.T) {
	if got := Run(Allocated(3), true); got != 25 {
		t.Fatalf("fast branch = %v, want 25", got)
	}
	if got := Run(nil, false); got != 22 {
		t.Fatalf("guarded branch = %v, want 22", got)
	}
}
`)
}
