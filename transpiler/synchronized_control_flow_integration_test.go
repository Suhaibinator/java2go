package transpiler

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestSynchronizedControlFlow_RuntimeParity(t *testing.T) {
	src := `
public class SynchronizedControlProgram {
    private static final Object LOCK = new Object();
    private static final Object SECOND_LOCK = new Object();
    private static int returnAudit = 0;
	private static int voidReturnAudit = 0;

    public static int outerContinueAndBreak() {
        int trace = 0;
        for (int index = 0; index < 3; index++) {
            synchronized (LOCK) {
                trace = trace * 10 + index + 1;
                if (index == 0) {
                    continue;
                }
                break;
            }
        }
        synchronized (LOCK) {
            trace = trace * 10 + 8;
        }
        return trace;
    }

    public static int labeledOuterTransfers() {
        int trace = 0;
        outer:
        for (int row = 0; row < 3; row++) {
            for (int column = 0; column < 2; column++) {
                synchronized (LOCK) {
                    trace = trace * 10 + row + 1;
                    if (row == 0) {
                        continue outer;
                    }
                    break outer;
                }
            }
        }
        synchronized (LOCK) {
            trace = trace * 10 + 8;
        }
        return trace;
    }

    public static int localLoopTargets() {
        int trace = 0;
        synchronized (LOCK) {
            for (int index = 0; index < 4; index++) {
                if (index == 0) {
                    continue;
                }
                trace = trace * 10 + index;
                if (index == 2) {
                    break;
                }
            }
            trace = trace * 10 + 7;
        }
        return trace;
    }

    public static int directReturn() {
        synchronized (LOCK) {
            return 31;
        }
    }

	private static void directVoidReturn() {
		synchronized (LOCK) {
			voidReturnAudit = 3;
			return;
		}
		voidReturnAudit = 99;
	}

	public static int probeAfterVoidReturn() {
		directVoidReturn();
		synchronized (LOCK) {
			return voidReturnAudit * 10 + 8;
		}
	}

    public static int returnRunsFinally() {
        synchronized (LOCK) {
            try {
                return 41;
            } finally {
                returnAudit = 7;
            }
        }
    }

    public static int probeAfterReturn() {
        synchronized (LOCK) {
            return returnAudit;
        }
    }

    public static int outerFinallyAfterReturn() {
        try {
            synchronized (LOCK) {
                return 42;
            }
        } finally {
            returnAudit = returnAudit * 10 + 9;
        }
    }

    public static int throwAndReenter() {
        int trace = 0;
        try {
            synchronized (LOCK) {
                trace = 1;
                throw new IllegalStateException();
            }
        } catch (IllegalStateException expected) {
            trace = trace * 10 + 2;
        }
        synchronized (LOCK) {
            trace = trace * 10 + 3;
        }
        return trace;
    }

    private static int throwingReturnValue() {
        throw new IllegalStateException();
    }

    public static int returnExpressionThrowAndReenter() {
        try {
            synchronized (LOCK) {
                return throwingReturnValue();
            }
        } catch (IllegalStateException expected) {
            synchronized (LOCK) {
                return 58;
            }
        }
    }

    public static int breakTargetsOuterSwitch() {
        int trace = 0;
        switch (1) {
            case 1:
                synchronized (LOCK) {
                    trace = 1;
                    break;
                }
            default:
                trace = 99;
        }
        synchronized (LOCK) {
            trace = trace * 10 + 8;
        }
        return trace;
    }

    public static int outerFinallyAfterContinue() {
        int trace = 0;
        for (int index = 0; index < 3; index++) {
            try {
                synchronized (LOCK) {
                    trace = trace * 10 + index + 1;
                    continue;
                }
            } finally {
                trace = trace * 10 + 7;
            }
        }
        synchronized (LOCK) {
            trace = trace * 10 + 8;
        }
        return trace;
    }

    public static int finallyReturnOverridesContinue() {
        for (int index = 0; index < 1; index++) {
            synchronized (LOCK) {
                try {
                    continue;
                } finally {
                    return 51;
                }
            }
        }
        return -1;
    }

    public static int finallyContinueOverridesReturn() {
        int trace = 0;
        for (int index = 0; index < 2; index++) {
            synchronized (LOCK) {
                try {
                    return -1;
                } finally {
                    trace = trace * 10 + index + 1;
                    continue;
                }
            }
        }
        return trace;
    }

    public static int finallyThrowOverridesContinue() {
        int trace = 0;
        try {
            for (int index = 0; index < 1; index++) {
                synchronized (LOCK) {
                    try {
                        continue;
                    } finally {
                        throw new IllegalStateException();
                    }
                }
            }
        } catch (IllegalStateException expected) {
            trace = 6;
        }
        synchronized (LOCK) {
            trace = trace * 10 + 8;
        }
        return trace;
    }

    public static int finallyBreakOverridesThrow() {
        int trace = 0;
        for (int index = 0; index < 1; index++) {
            synchronized (LOCK) {
                try {
                    throw new IllegalStateException();
                } finally {
                    trace = 7;
                    break;
                }
            }
        }
        synchronized (LOCK) {
            trace = trace * 10 + 8;
        }
        return trace;
    }

    public static int nestedMonitorContinue() {
        int trace = 0;
        for (int index = 0; index < 2; index++) {
            synchronized (LOCK) {
                synchronized (SECOND_LOCK) {
                    trace = trace * 10 + index + 1;
                    continue;
                }
            }
        }
        synchronized (LOCK) {
            synchronized (SECOND_LOCK) {
                trace = trace * 10 + 8;
            }
        }
        return trace;
    }

    public static int labeledBlockBreak() {
        int trace = 0;
        target: {
            synchronized (LOCK) {
                trace = 1;
                break target;
            }
            trace = 99;
        }
        synchronized (LOCK) {
            trace = trace * 10 + 8;
        }
        return trace;
    }

    public static int directlyLabeledSynchronizedBreak() {
        int trace = 0;
        target:
        synchronized (LOCK) {
            trace = 1;
            break target;
        }
        synchronized (LOCK) {
            trace = trace * 10 + 8;
        }
        return trace;
    }

    public static int doWhileContinue() {
        int trace = 0;
        int index = 0;
        do {
            synchronized (LOCK) {
                index++;
                trace = trace * 10 + index;
                continue;
            }
        } while (index < 2);
        synchronized (LOCK) {
            trace = trace * 10 + 8;
        }
        return trace;
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "UNSUPPORTED") {
		t.Fatalf("expected synchronized abrupt-control lowering without placeholders, got:\n%s", out)
	}

	runGeneratedWithStdjava(t, out, `
package main

import (
    "testing"
    "time"
)

func TestSynchronizedControlParity(t *testing.T) {
    tests := []struct {
        name string
        run  func() int32
        want int32
    }{
        {"outer continue and break", OuterContinueAndBreak, 128},
        {"labeled outer transfers", LabeledOuterTransfers, 128},
        {"local loop targets", LocalLoopTargets, 127},
        {"direct return", DirectReturn, 31},
        {"direct void return and re-enter", ProbeAfterVoidReturn, 38},
        {"return runs finally", ReturnRunsFinally, 41},
        {"probe after return", ProbeAfterReturn, 7},
        {"outer finally after return", OuterFinallyAfterReturn, 42},
        {"probe after outer finally return", ProbeAfterReturn, 79},
        {"throw and re-enter", ThrowAndReenter, 123},
        {"return expression throw and re-enter", ReturnExpressionThrowAndReenter, 58},
        {"break targets outer switch", BreakTargetsOuterSwitch, 18},
        {"outer finally after continue", OuterFinallyAfterContinue, 1727378},
        {"finally return overrides continue", FinallyReturnOverridesContinue, 51},
        {"finally continue overrides return", FinallyContinueOverridesReturn, 12},
        {"finally throw overrides continue", FinallyThrowOverridesContinue, 68},
        {"finally break overrides throw", FinallyBreakOverridesThrow, 78},
        // LOCK and SECOND_LOCK are distinct Java objects. Same-object monitor
        // reentrancy is intentionally outside this control-flow regression test.
        {"nested distinct-monitor continue", NestedMonitorContinue, 128},
        {"labeled block break", LabeledBlockBreak, 18},
        {"directly labeled synchronized break", DirectlyLabeledSynchronizedBreak, 18},
        {"do-while continue", DoWhileContinue, 128},
    }
    for _, test := range tests {
        result := make(chan int32, 1)
        go func(run func() int32) {
            result <- run()
        }(test.run)
        select {
        case got := <-result:
            if got != test.want {
                t.Errorf("%s: got %d, want %d", test.name, got, test.want)
            }
        case <-time.After(3 * time.Second):
            t.Fatalf("%s retained a synchronized monitor", test.name)
        }
    }
}
`)
}

func TestSynchronizedControlFlow_GeneratedNamesAvoidJavaLocals(t *testing.T) {
	const template = `
public class SynchronizedNameCollisionProgram {
    private static final Object LOCK = new Object();

    public static int run() {
        int trace = 0;
        for (int index = 0; index < 2; index++) {
            synchronized (LOCK) {
                int MONITOR_NAME = 4;
                trace = trace * 10 + MONITOR_NAME;
                if (index == 0) {
                    continue;
                }
            }
            int RETURN_FLAG_NAME = 1;
            int RETURN_VALUE_NAME = 2;
            int CONTROL_NAME = 3;
            trace = trace * 10 + RETURN_FLAG_NAME + RETURN_VALUE_NAME + CONTROL_NAME;
        }
        return trace;
    }
}
`

	baseline := renderGoFileFromJava(t, template)
	generatedName := func(pattern string) string {
		t.Helper()
		name := regexp.MustCompile(pattern).FindString(baseline)
		if name == "" {
			t.Fatalf("baseline synchronized lowering did not contain %q:\n%s", pattern, baseline)
		}
		return name
	}
	collisionSource := strings.NewReplacer(
		"MONITOR_NAME", generatedName(`__java2goMonitor_\d+`),
		"RETURN_FLAG_NAME", generatedName(`__java2goSyncShouldReturn_\d+`),
		"RETURN_VALUE_NAME", generatedName(`__java2goSyncReturnValue_\d+`),
		"CONTROL_NAME", generatedName(`__java2goSyncControl_\d+`),
	).Replace(template)

	out := renderGoFileFromJava(t, collisionSource)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestSynchronizedGeneratedNameHygiene(t *testing.T) {
    if got := Run(); got != 446 {
        t.Fatalf("Run() = %d, want 446", got)
    }
}
`)
}

func TestSynchronizedControlFlow_SiblingGeneratedNamesDoNotReachFixedPoint(t *testing.T) {
	const template = `
public class SynchronizedSiblingNameProgram {
    private static final Object LOCK = new Object();

    public static int run() {
        int trace = 0;
        // Padding before the first statement makes appending one decimal zero
        // to its source offset comfortably later than this method's base size.
        /*                                                                                                    */
        synchronized (LOCK) {
            int FIRST_FLAG_COLLISION = 1;
            trace = FIRST_FLAG_COLLISION;
        }
        /*FILLER*/
        synchronized (LOCK) {
            trace = trace * 10 + 2;
        }
        return trace;
    }
}
`

	monitorStarts := func(generated string) []int {
		t.Helper()
		matches := regexp.MustCompile(`__java2goMonitor_(\d+)`).FindAllStringSubmatch(generated, -1)
		starts := make([]int, 0, len(matches))
		seen := make(map[int]struct{})
		for _, match := range matches {
			start, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatalf("parse synchronized source offset %q: %v", match[1], err)
			}
			if _, duplicate := seen[start]; duplicate {
				continue
			}
			seen[start] = struct{}{}
			starts = append(starts, start)
		}
		return starts
	}

	baseline := renderGoFileFromJava(t, template)
	baselineStarts := monitorStarts(baseline)
	if len(baselineStarts) != 2 {
		t.Fatalf("baseline emitted %d monitor names, want 2:\n%s", len(baselineStarts), baseline)
	}
	firstStart := baselineStarts[0]
	firstCollision := "__java2goSyncShouldReturn_" + strconv.Itoa(firstStart)

	withoutFiller := strings.NewReplacer(
		"FIRST_FLAG_COLLISION", firstCollision,
		"/*FILLER*/", "",
	).Replace(template)
	shiftedStarts := monitorStarts(renderGoFileFromJava(t, withoutFiller))
	if len(shiftedStarts) != 2 || shiftedStarts[0] != firstStart {
		t.Fatalf("collision setup changed first synchronized offset: got %v, want first %d", shiftedStarts, firstStart)
	}
	targetSecondStart, err := strconv.Atoi(strconv.Itoa(firstStart) + "0")
	if err != nil {
		t.Fatalf("build sibling fixed-point offset: %v", err)
	}
	fillerLength := targetSecondStart - shiftedStarts[1]
	if fillerLength < 0 {
		t.Fatalf("test source is too large for fixed-point setup: second starts at %d, target %d", shiftedStarts[1], targetSecondStart)
	}
	collisionSource := strings.NewReplacer(
		"FIRST_FLAG_COLLISION", firstCollision,
		"/*FILLER*/", strings.Repeat(" ", fillerLength),
	).Replace(template)
	finalStarts := monitorStarts(renderGoFileFromJava(t, collisionSource))
	if len(finalStarts) != 2 || finalStarts[1] != targetSecondStart {
		t.Fatalf("failed to build sibling fixed-point offsets: got %v, want second %d", finalStarts, targetSecondStart)
	}

	out := renderGoFileFromJava(t, collisionSource)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestSiblingGeneratedNameHygiene(t *testing.T) {
    if got := Run(); got != 12 {
        t.Fatalf("Run() = %d, want 12", got)
    }
}
`)
}

func TestSynchronizedControlFlow_ConstructorReturnParity(t *testing.T) {
	out := renderGoFileFromJava(t, `
public class SynchronizedConstructorReturnProgram {
    private static final Object LOCK = new Object();
    private int trace;

    public SynchronizedConstructorReturnProgram(int mode) {
        trace = 1;
        if (mode == 0) {
            return;
        }
        if (mode == 3) {
            try {
                return;
            } finally {
                trace = 5;
            }
        }
        synchronized (LOCK) {
            trace = 2;
            if (mode == 1) {
                return;
            }
            trace = 3;
        }
        trace = 4;
    }

    public static int run() {
        SynchronizedConstructorReturnProgram direct = new SynchronizedConstructorReturnProgram(0);
        SynchronizedConstructorReturnProgram synced = new SynchronizedConstructorReturnProgram(1);
        SynchronizedConstructorReturnProgram normal = new SynchronizedConstructorReturnProgram(2);
        SynchronizedConstructorReturnProgram finalized = new SynchronizedConstructorReturnProgram(3);
        return direct.trace * 1000 + synced.trace * 100 + normal.trace * 10 + finalized.trace;
    }
}
`)

	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestConstructorReturnParity(t *testing.T) {
    if got := Run(); got != 1245 {
        t.Fatalf("Run() = %d, want 1245", got)
    }
}
`)
}

func TestSynchronizedControlFlow_LambdaReturnContextParity(t *testing.T) {
	out := renderGoFileFromJava(t, `
interface SynchronizedVoidAction {
    void run();
}

interface SynchronizedIntSupplier {
    int get();
}

public class SynchronizedLambdaReturnProgram {
    private static final Object LOCK = new Object();
    private static int trace;

    private static void appendSupplierValue() {
        SynchronizedIntSupplier supplier = () -> {
            synchronized (LOCK) {
                return 7;
            }
        };
        trace = trace * 10 + supplier.get();
    }

    public static int run() {
        trace = 0;
        SynchronizedVoidAction action = () -> {
            synchronized (LOCK) {
                trace = 3;
                return;
            }
        };
        action.run();
        appendSupplierValue();
        return trace;
    }
}
`)

	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestLambdaReturnContextParity(t *testing.T) {
    if got := Run(); got != 37 {
        t.Fatalf("Run() = %d, want 37", got)
    }
}
`)
}
