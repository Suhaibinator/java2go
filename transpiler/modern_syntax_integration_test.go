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

// --- Text blocks (Java 13+) ---

func TestTextBlock_IncidentalWhitespaceStripping(t *testing.T) {
	src := "package modern;\n" +
		"public class App {\n" +
		"    public static String run() {\n" +
		"        return \"\"\"\n" +
		"                Hello\n" +
		"                  World\n" +
		"                Bye\"\"\";\n" +
		"    }\n" +
		"}\n"

	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected transpiler to produce non-empty Go output")
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestTextBlockRuntime(t *testing.T) {
	got := Run()
	want := "Hello\n  World\nBye"
	if got != want {
		t.Fatalf("Run() = %q, want %q", got, want)
	}
}
`)
}

func TestTextBlock_TrailingNewlineWhenClosingOnOwnLine(t *testing.T) {
	src := "package modern;\n" +
		"public class App {\n" +
		"    public static String run() {\n" +
		"        return \"\"\"\n" +
		"                line1\n" +
		"                line2\n" +
		"                \"\"\";\n" +
		"    }\n" +
		"}\n"

	out := renderGoFileFromJava(t, src)

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestTextBlockTrailingNewline(t *testing.T) {
	got := Run()
	want := "line1\nline2\n"
	if got != want {
		t.Fatalf("Run() = %q, want %q", got, want)
	}
}
`)
}

// --- Records (Java 14+) ---

func TestRecord_StructAccessorsConstructor(t *testing.T) {
	src := `
package modern;
public record Point(int x, int y) {
    public int sum() { return x + y; }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)

	if !strings.Contains(flat, "type Point struct { x int32 y int32 }") {
		t.Fatalf("expected record components to become unexported struct fields, got:\n%s", out)
	}
	if !strings.Contains(flat, "func NewPoint(x int32, y int32) *Point") {
		t.Fatalf("expected a canonical record constructor, got:\n%s", out)
	}
	if !strings.Contains(flat, "func (pt *Point) X() int32 { return pt.x }") {
		t.Fatalf("expected an exported accessor named after the component, got:\n%s", out)
	}
	if !strings.Contains(flat, "func (pt *Point) Equals(other *Point) bool") {
		t.Fatalf("expected a synthesized value-equality method, got:\n%s", out)
	}
}

func TestRecord_RuntimeBehavior(t *testing.T) {
	src := `
package modern;
public class App {
    record Point(int x, int y) {
        int sum() { return x + y; }
    }
    public static int run() {
        Point p = new Point(3, 4);
        Point q = new Point(3, 4);
        Point r = new Point(5, 6);
        int eq = (p.equals(q) ? 1 : 0) + (p.equals(r) ? 1 : 0);
        return p.x() * 1000 + p.y() * 100 + p.sum() * 10 + eq;
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

func TestRecordRuntime(t *testing.T) {
	got := Run()
	// x=3, y=4, sum=7, eq = (p==q ->1) + (p==r ->0) = 1
	// 3*1000 + 4*100 + 7*10 + 1 = 3471
	if got != 3471 {
		t.Fatalf("Run() = %d, want 3471", got)
	}
}
`)
}
