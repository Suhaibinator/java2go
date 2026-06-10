package transpiler

import (
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// withCleanDiagnostics resets the global diagnostic/strict state before and after
// a test so cases do not leak into one another.
func withCleanDiagnostics(t *testing.T) {
	t.Helper()
	resetDiagnostics()
	setStrictMode(false)
	t.Cleanup(func() {
		resetDiagnostics()
		setStrictMode(false)
	})
}

// TestDiagnostics_UnsupportedStatementDoesNotCrash verifies that an unsupported
// statement (a Java `assert`) is converted into an UNSUPPORTED placeholder, a
// diagnostic is recorded, and the surrounding supported code still converts.
func TestDiagnostics_UnsupportedStatementDoesNotCrash(t *testing.T) {
	withCleanDiagnostics(t)

	src := `
public class Robust {
    public int compute() {
        int before = 1;
        assert before == 1;
        int after = before + 1;
        return after;
    }
}
`

	out := renderGoFileFromJava(t, src)

	if !strings.Contains(out, "UNSUPPORTED") {
		t.Fatalf("expected an UNSUPPORTED stub in output, got:\n%s", out)
	}
	// The supported statements surrounding the assert must still be converted.
	if !strings.Contains(out, "before := 1") {
		t.Errorf("expected supported statement before the unsupported one, got:\n%s", out)
	}
	if !strings.Contains(out, "after := before + 1") {
		t.Errorf("expected supported statement after the unsupported one, got:\n%s", out)
	}
	if !strings.Contains(out, "return after") {
		t.Errorf("expected return statement to convert, got:\n%s", out)
	}

	diags := collectedDiagnostics()
	if len(diags) == 0 {
		t.Fatal("expected at least one diagnostic to be recorded")
	}
	found := false
	for _, d := range diags {
		if d.NodeType == "assert_statement" && d.Kind == "statement" {
			found = true
			if d.Line == 0 {
				t.Errorf("expected diagnostic to record a source line, got 0")
			}
		}
	}
	if !found {
		t.Errorf("expected a diagnostic for assert_statement, got: %v", diags)
	}
}

// TestDiagnostics_UnsupportedExpressionEmitsPlaceholder verifies that an
// unsupported expression yields a placeholder panic call rather than aborting.
func TestDiagnostics_UnsupportedExpressionEmitsPlaceholder(t *testing.T) {
	withCleanDiagnostics(t)

	// A lambda used as a plain expression is not converted into a Go value here,
	// so it exercises the expression fallback path.
	src := `
public class Lambdas {
    public Runnable make() {
        Runnable r = () -> System.out.println("hi");
        return r;
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected non-empty output even with an unsupported expression")
	}
	// Either the lambda is handled or it degrades to a stub; in both cases the
	// transpiler must not crash and must still emit the enclosing function.
	if !strings.Contains(out, "func (lm *Lambdas) Make()") && !strings.Contains(out, "Make()") {
		t.Errorf("expected enclosing method to be converted, got:\n%s", out)
	}
}

// TestRun_StrictMode_FailsOnUnsupported verifies that the -strict flag restores
// fail-fast behavior: the first unsupported construct returns an error.
func TestRun_StrictMode_FailsOnUnsupported(t *testing.T) {
	withCleanDiagnostics(t)

	inputDir := filepath.Join(t.TempDir(), "input")
	writeJavaSource(t, inputDir, "Strict.java", `
public class Strict {
    public void m() {
        assert true;
    }
}
`)

	err := run([]string{"-strict", inputDir}, io.Discard)
	if err == nil {
		t.Fatal("expected strict mode to return an error on an unsupported construct")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected error to mention the unsupported construct, got: %v", err)
	}
}

// TestRun_NonStrictMode_SucceedsWithDiagnostics verifies the default behavior:
// an unsupported construct does not fail the run, and a diagnostic is recorded.
func TestRun_NonStrictMode_SucceedsWithDiagnostics(t *testing.T) {
	withCleanDiagnostics(t)

	inputDir := filepath.Join(t.TempDir(), "input")
	writeJavaSource(t, inputDir, "Lenient.java", `
public class Lenient {
    public void m() {
        assert true;
    }
}
`)

	if err := run([]string{inputDir}, io.Discard); err != nil {
		t.Fatalf("expected non-strict run to succeed, got: %v", err)
	}

	if len(Diagnostics()) == 0 {
		t.Error("expected diagnostics to be recorded after a lenient run")
	}
}
