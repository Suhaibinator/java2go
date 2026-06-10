package transpiler

import (
	"strings"
	"testing"
)

// --- Switch expressions (Java 12+) ---

func TestSwitchExpression_ArrowFormWithYield(t *testing.T) {
	src := `
package modern;
public class App {
    public static int classify(int x) {
        return switch (x) {
            case 1, 2 -> 10;
            case 3 -> { yield 30; }
            default -> 0;
        };
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "return func() int32 {") {
		t.Fatalf("expected switch expression to lower to an IIFE, got:\n%s", out)
	}
	if !strings.Contains(flat, "case 1, 2:") || !strings.Contains(flat, "return 10") {
		t.Fatalf("expected multi-label arm to return its value, got:\n%s", out)
	}
	if !strings.Contains(flat, "case 3:") || !strings.Contains(flat, "return 30") {
		t.Fatalf("expected yield to lower to return, got:\n%s", out)
	}
}

func TestSwitchExpression_RuntimeBehavior(t *testing.T) {
	src := `
package modern;
public class App {
    static String dayKind(int day) {
        return switch (day) {
            case 1, 2, 3, 4, 5 -> "weekday";
            case 6, 7 -> "weekend";
            default -> "invalid";
        };
    }
    public static String run() {
        return dayKind(3) + "," + dayKind(7) + "," + dayKind(9);
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected transpiler to produce non-empty Go output")
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestSwitchExprRuntime(t *testing.T) {
	got := Run()
	if got != "weekday,weekend,invalid" {
		t.Fatalf("Run() = %q, want %q", got, "weekday,weekend,invalid")
	}
}
`)
}

func TestSwitchExpression_StringYieldBlock_RuntimeBehavior(t *testing.T) {
	src := `
package modern;
public class App {
    static int score(String grade) {
        int points = switch (grade) {
            case "A" -> 4;
            case "B" -> 3;
            default -> {
                yield 0;
            }
        };
        return points;
    }
    public static int run() {
        return score("A") * 100 + score("B") * 10 + score("F");
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected transpiler to produce non-empty Go output")
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestSwitchScoreRuntime(t *testing.T) {
	got := Run()
	// score("A")=4, score("B")=3, score("F")=0 -> 4*100 + 3*10 + 0 = 430
	if got != 430 {
		t.Fatalf("Run() = %d, want 430", got)
	}
}
`)
}

// --- instanceof pattern matching (Java 16+) ---

func TestInstanceofPattern_BindsVariable(t *testing.T) {
	src := `
package modern;
public class App {
    public static String describe(Object o) {
        if (o instanceof String s) {
            return "str:" + s;
        }
        return "other";
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "if s, ok := any(o).(string); ok {") {
		t.Fatalf("expected instanceof pattern to lower to a type-assertion if-init, got:\n%s", out)
	}
}

func TestInstanceofPattern_RuntimeBehavior(t *testing.T) {
	// Uses String pattern binding plus a non-matching reference type so the bound
	// variable is exercised at runtime without depending on int autoboxing (which
	// is a separate semantic-fidelity concern).
	src := `
package modern;
public class App {
    static String describe(Object o) {
        if (o instanceof String s) {
            return "string:" + s + ":" + s.length();
        } else {
            return "other";
        }
    }
    public static String run() {
        Object a = "hi";
        String[] arr = new String[2];
        Object b = arr;
        return describe(a) + "|" + describe(b);
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected transpiler to produce non-empty Go output")
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestInstanceofPatternRuntime(t *testing.T) {
	got := Run()
	if got != "string:hi:2|other" {
		t.Fatalf("Run() = %q, want %q", got, "string:hi:2|other")
	}
}
`)
}
