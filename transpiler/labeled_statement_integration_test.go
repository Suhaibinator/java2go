package transpiler

import (
	"strings"
	"testing"
)

func TestLabeledStatement_NonLoopBreakRuntimeParity(t *testing.T) {
	src := `
public class LabeledStatementProgram {
    public static int directBlock() {
        int trace = 0;
        exit: {
            trace = 1;
            if (trace == 1) {
                break exit;
            }
            trace = 99;
        }
        return trace * 10 + 2;
    }

    public static int nestedBlocks() {
        int trace = 0;
        outer: {
            trace = 1;
            inner: {
                trace = trace * 10 + 2;
                if (trace == 12) {
                    break inner;
                }
                trace = 99;
            }
            trace = trace * 10 + 3;
            if (trace == 123) {
                break outer;
            }
            trace = 99;
        }
        return trace;
    }

    public static int continueEnclosingLoop() {
        int trace = 0;
        for (int index = 0; index < 3; index++) {
            boundary: {
                if (index == 0) {
                    trace = 1;
                    continue;
                }
                trace = trace * 10 + 2;
                break boundary;
            }
            trace = trace * 10 + 3;
        }
        return trace;
    }

    public static int breakEnclosingLoop() {
        int trace = 0;
        for (int index = 0; index < 3; index++) {
            boundary: {
                trace = trace * 10 + 1;
                break;
            }
            trace = 99;
        }
        return trace;
    }

    public static int labeledLoopStillUsesBreak() {
        int trace = 0;
        outer: for (int row = 0; row < 3; row++) {
            for (int column = 0; column < 3; column++) {
                trace++;
                if (column == 1) {
                    continue outer;
                }
            }
            trace = 99;
        }
        return trace;
    }

    public static int labeledSwitchStillUsesBreak() {
        int trace = 0;
        exit: switch (2) {
            case 1:
                trace = 1;
                break;
            case 2:
                trace = 7;
                break exit;
            default:
                trace = 9;
        }
        return trace;
    }

    public static int throughFinally() {
        int trace = 0;
        exit: {
            try {
                trace = 1;
                break exit;
            } finally {
                trace = trace * 10 + 2;
            }
            trace = 99;
        }
        return trace;
    }

    public static int throughNestedFinally() {
        int trace = 0;
        exit: {
            try {
                try {
                    trace = 1;
                    break exit;
                } finally {
                    trace = trace * 10 + 2;
                }
            } finally {
                trace = trace * 10 + 3;
            }
        }
        return trace;
    }

    public static int labelOnIf() {
        int trace = 0;
        exit: if (true) {
            trace = 3;
            if (trace == 3) {
                break exit;
            }
            trace = 99;
        }
        return trace * 10 + 4;
    }

    public static int labelOnTry() {
        int trace = 0;
        exit: try {
            trace = 5;
            break exit;
        } finally {
            trace = trace * 10 + 6;
        }
        return trace;
    }

    public static int variableScope() {
        int result = 0;
        scope: {
            int local = 7;
            result = local;
            if (result == 7) {
                break scope;
            }
            result = 99;
        }
        int local = 5;
        return result * 10 + local;
    }

    public static int sanitizedLabelCollision() {
        int trace = 0;
        map: {
            trace = 1;
            map_: for (int index = 0; index < 2; index++) {
                trace = trace * 10 + 2;
                break map_;
            }
            trace = trace * 10 + 3;
            if (trace == 123) {
                break map;
            }
            trace = 99;
        }
        return trace;
    }

    public static int twoLoopSanitizedLabelCollision() {
        int trace = 0;
        map: for (int outer = 0; outer < 2; outer++) {
            trace = trace * 10 + 1;
            map_: for (int inner = 0; inner < 2; inner++) {
                trace = trace * 10 + 2;
                if (inner == 0) {
                    continue map_;
                }
                break map;
            }
            trace = 99;
        }
        return trace;
    }

}
`

	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "UNSUPPORTED") {
		t.Fatalf("expected arbitrary labeled statements to lower without placeholders, got:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestLabeledStatementParity(t *testing.T) {
    tests := []struct {
        name string
        got  int32
        want int32
    }{
        {"direct block", DirectBlock(), 12},
        {"nested blocks", NestedBlocks(), 123},
        {"continue enclosing loop", ContinueEnclosingLoop(), 12323},
        {"break enclosing loop", BreakEnclosingLoop(), 1},
        {"labeled loop", LabeledLoopStillUsesBreak(), 6},
        {"labeled switch", LabeledSwitchStillUsesBreak(), 7},
        {"through finally", ThroughFinally(), 12},
        {"through nested finally", ThroughNestedFinally(), 123},
		{"label on if", LabelOnIf(), 34},
		{"label on try", LabelOnTry(), 56},
		{"variable scope", VariableScope(), 75},
		{"sanitized label collision", SanitizedLabelCollision(), 123},
		{"two-loop sanitized label collision", TwoLoopSanitizedLabelCollision(), 122},
	}
    for _, test := range tests {
        if test.got != test.want {
            t.Errorf("%s: got %d, want %d", test.name, test.got, test.want)
        }
    }
}
`)
}
