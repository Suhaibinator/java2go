package transpiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot walks up from the test's working directory to the module root (the
// directory holding go.mod for github.com/NickyBoy89/java2go) so generated test
// modules can point a replace directive at the real stdjava package.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if data, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
			if strings.Contains(string(data), "module github.com/NickyBoy89/java2go") {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repository root containing go.mod")
		}
		dir = parent
	}
}

// runGeneratedWithStdjava compiles the transpiled Go source together with a
// behavior test in a temporary module whose stdjava dependency is replaced with
// the in-repo package, then runs `go test`. It is the exception-suite analogue
// of runGoTestInTempModule for code that imports the stdjava runtime.
func runGeneratedWithStdjava(t *testing.T, generatedGo, goTestSource string) {
	t.Helper()
	root := repoRoot(t)
	tempDir := t.TempDir()

	goMod := "module generated\n\ngo 1.26.0\n\n" +
		"require github.com/NickyBoy89/java2go v0.0.0\n\n" +
		"replace github.com/NickyBoy89/java2go => " + root + "\n"
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "generated.go"), []byte(generatedGo), 0644); err != nil {
		t.Fatalf("write generated.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "generated_behavior_test.go"), []byte(goTestSource), 0644); err != nil {
		t.Fatalf("write behavior test: %v", err)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = tempDir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed:\n%s", out)
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tempDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated code behavior verification failed:\n%s", out)
	}
}

func TestExceptions_CatchBySupertype(t *testing.T) {
	src := `
public class ExProgram {
    public static String run(int n) {
        try {
            if (n < 0) {
                throw new IllegalArgumentException("negative input");
            }
            return "ok";
        } catch (RuntimeException e) {
            return "caught:" + e.getMessage();
        }
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "stdjava.NewIllegalArgumentException(") {
		t.Fatalf("expected stdjava constructor for thrown exception, got:\n%s", out)
	}
	if !strings.Contains(out, `stdjava.CaughtAs(`) {
		t.Fatalf("expected catch dispatch via stdjava.CaughtAs, got:\n%s", out)
	}

	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestRun(t *testing.T) {
	if got := Run(-1); got != "caught:negative input" {
		t.Fatalf("catch-by-supertype: got %q", got)
	}
	if got := Run(1); got != "ok" {
		t.Fatalf("no-throw path: got %q", got)
	}
}
`)
}

func TestExceptions_MultiCatch(t *testing.T) {
	src := `
public class MultiProgram {
    public static String run(int n) {
        try {
            if (n == 1) {
                throw new IllegalStateException("state");
            }
            if (n == 2) {
                throw new NumberFormatException("number");
            }
            return "ok";
        } catch (IllegalStateException | NumberFormatException e) {
            return "multi:" + e.getMessage();
        }
    }
}
`
	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestRun(t *testing.T) {
	if got := Run(1); got != "multi:state" {
		t.Fatalf("multi-catch first type: got %q", got)
	}
	if got := Run(2); got != "multi:number" {
		t.Fatalf("multi-catch second type: got %q", got)
	}
	if got := Run(0); got != "ok" {
		t.Fatalf("no-throw path: got %q", got)
	}
}
`)
}

func TestExceptions_FinallyOrderingAndRethrow(t *testing.T) {
	src := `
public class FinallyProgram {
    public static String run(boolean rethrow) {
        String trace = "";
        try {
            trace = trace + "try;";
            try {
                throw new ArithmeticException("boom");
            } catch (ArithmeticException e) {
                trace = trace + "catch;";
                if (rethrow) {
                    throw new IllegalStateException("rethrown");
                }
            } finally {
                trace = trace + "finally;";
            }
            trace = trace + "after;";
        } catch (IllegalStateException e) {
            trace = trace + "outer:" + e.getMessage() + ";";
        }
        return trace;
    }
}
`
	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestRun(t *testing.T) {
	// Finally must run before control leaves the inner try, and the outer
	// catch must see the rethrown IllegalStateException.
	if got := Run(true); got != "try;catch;finally;outer:rethrown;" {
		t.Fatalf("rethrow ordering: got %q", got)
	}
	if got := Run(false); got != "try;catch;finally;after;" {
		t.Fatalf("normal ordering: got %q", got)
	}
}
`)
}

func TestExceptions_UserDefinedHierarchy(t *testing.T) {
	src := `
public class UserExProgram {
    static class AppException extends RuntimeException {
        public AppException(String m) { super(m); }
    }
    static class NotFoundException extends AppException {
        public NotFoundException(String m) { super(m); }
    }
    public static String run(int n) {
        try {
            if (n == 1) {
                throw new NotFoundException("missing");
            }
            if (n == 2) {
                throw new AppException("generic");
            }
            return "ok";
        } catch (AppException e) {
            return "app:" + e.getMessage();
        }
    }
    public static String onlyNotFound(int n) {
        try {
            if (n == 1) {
                throw new AppException("generic");
            }
            return "ok";
        } catch (NotFoundException e) {
            return "nf:" + e.getMessage();
        }
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, `stdjava.RegisterException(`) {
		t.Fatalf("expected user exception registration, got:\n%s", out)
	}
	if !strings.Contains(out, "ThrowableTypeName() string") {
		t.Fatalf("expected ThrowableTypeName override on user exception, got:\n%s", out)
	}

	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestRun(t *testing.T) {
	// A subclass instance is caught by a clause for its supertype.
	if got := Run(1); got != "app:missing" {
		t.Fatalf("subclass caught by supertype: got %q", got)
	}
	if got := Run(2); got != "app:generic" {
		t.Fatalf("exact user type: got %q", got)
	}
}

func TestOnlyNotFound(t *testing.T) {
	// A supertype instance must NOT be caught by a subtype clause; it escapes
	// as a panic.
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected AppException to escape NotFoundException catch")
		}
	}()
	OnlyNotFound(1)
}
`)
}

func TestExceptions_NativeRuntimePanicNormalization(t *testing.T) {
	// Native Go runtime panics (divide by zero, nil dereference, index out of
	// range, failed cast) must be catchable as the corresponding Java exception.
	src := `
public class RuntimeExProgram {
    public static int divide(int a, int b) {
        try {
            return a / b;
        } catch (ArithmeticException e) {
            return -1;
        }
    }
    public static int indexAccess(int[] arr, int i) {
        try {
            return arr[i];
        } catch (ArrayIndexOutOfBoundsException e) {
            return -1;
        }
    }
    public static int byRuntimeException(int a, int b) {
        try {
            return a / b;
        } catch (RuntimeException e) {
            return -2;
        }
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "stdjava.NormalizePanic(") {
		t.Fatalf("expected recover boundary to call stdjava.NormalizePanic, got:\n%s", out)
	}

	runGeneratedWithStdjava(t, out, `
package main

import (
	"testing"

	"github.com/NickyBoy89/java2go/stdjava"
)

func TestDivide(t *testing.T) {
	if got := Divide(10, 2); got != 5 {
		t.Fatalf("Divide(10,2) = %d, want 5", got)
	}
	// Divide by zero panics in Go; it must be caught as ArithmeticException.
	if got := Divide(10, 0); got != -1 {
		t.Fatalf("Divide(10,0) = %d, want -1 (ArithmeticException caught)", got)
	}
}

func TestIndexAccess(t *testing.T) {
	arr := stdjava.PrimitiveArrayLiteral[int32](stdjava.PrimitiveTypeID("int"), 1, 2, 3)
	if got := IndexAccess(arr, 1); got != 2 {
		t.Fatalf("in-range access = %d, want 2", got)
	}
	if got := IndexAccess(arr, 9); got != -1 {
		t.Fatalf("out-of-range = %d, want -1 (ArrayIndexOutOfBoundsException caught)", got)
	}
	if got := IndexAccess(arr, -1); got != -1 {
		t.Fatalf("negative index = %d, want -1 (ArrayIndexOutOfBoundsException caught)", got)
	}
	empty := stdjava.NewPrimitiveArray[int32](0, stdjava.PrimitiveTypeID("int"))
	if got := IndexAccess(empty, 0); got != -1 {
		t.Fatalf("empty-array access = %d, want -1 (ArrayIndexOutOfBoundsException caught)", got)
	}
}

func TestNullIndexAccessRemainsNullPointerException(t *testing.T) {
	defer func() {
		recovered := stdjava.NormalizePanic(recover())
		if !stdjava.CaughtAs(recovered, "NullPointerException") || stdjava.CaughtAs(recovered, "ArrayIndexOutOfBoundsException") {
			t.Fatalf("null array panic = %T (%v), want only NullPointerException", recovered, recovered)
		}
	}()
	IndexAccess(nil, 0)
	t.Fatal("null array access unexpectedly returned")
}

func TestByRuntimeException(t *testing.T) {
	// A normalized ArithmeticException is also catchable as RuntimeException.
	if got := ByRuntimeException(1, 0); got != -2 {
		t.Fatalf("got %d, want -2 (caught as RuntimeException)", got)
	}
}
`)
}

func TestExceptions_ErrorNotCaughtByExceptionClause(t *testing.T) {
	// catch (Exception e) must not catch an Error-level throw, but catch
	// (Throwable t) must. This mirrors Java's exception hierarchy.
	src := `
public class ErrorProgram {
    public static String onlyException(int n) {
        try {
            if (n == 1) {
                throw new AssertionError("boom");
            }
            return "ok";
        } catch (Exception e) {
            return "ex";
        }
    }
    public static String byThrowable(int n) {
        try {
            if (n == 1) {
                throw new AssertionError("boom");
            }
            return "ok";
        } catch (Throwable t) {
            return "thr";
        }
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, `stdjava.CaughtAs(`) {
		t.Fatalf("catch (Exception) should now dispatch through CaughtAs, got:\n%s", out)
	}

	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestByThrowable(t *testing.T) {
	if got := ByThrowable(1); got != "thr" {
		t.Fatalf("AssertionError should be caught by catch (Throwable): got %q", got)
	}
}

func TestOnlyException(t *testing.T) {
	// AssertionError must escape catch (Exception e) as a panic.
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("AssertionError must not be caught by catch (Exception e)")
		}
	}()
	OnlyException(1)
}
`)
}
