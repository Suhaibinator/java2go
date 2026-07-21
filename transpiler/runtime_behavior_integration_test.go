package transpiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGoTestInTempModule(t *testing.T, generatedGo string, goTestSource string) {
	t.Helper()

	tempDir := t.TempDir()

	// Generated code may import the stdjava runtime (try/catch lowering,
	// intrinsics), so resolve it against this repository's copy.
	goMod := "module generated\n\ngo 1.25.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => " + repoRootDir(t) + "\n"
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed writing temp go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "generated.go"), []byte(generatedGo), 0644); err != nil {
		t.Fatalf("failed writing generated source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "generated_behavior_test.go"), []byte(goTestSource), 0644); err != nil {
		t.Fatalf("failed writing generated behavior test: %v", err)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = tempDir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed:\n%s", string(out))
	}

	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = tempDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated code behavior verification failed:\n%s", string(out))
	}
}

func TestGeneratedCode_RuntimeBehavior_Arithmetic(t *testing.T) {
	src := `
public class RuntimeProgram {
    public static int calc() {
        return 2 + 3 + 4;
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

func TestCalcBehavior(t *testing.T) {
	got := Calc()
	if got != 9 {
		t.Fatalf("Calc() = %d, want 9", got)
	}
}
`)
}

func TestGeneratedCode_RuntimeBehavior_FunctionalInterfaceAdapter(t *testing.T) {
	src := `
public interface IntMapper {
    int map(int value);
}

public class RuntimeLambdaProgram {
    public static int run() {
        IntMapper plusOne = v -> v + 1;
        return plusOne.map(4);
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

func TestLambdaAdapterBehavior(t *testing.T) {
	got := Run()
	if got != 5 {
		t.Fatalf("Run() = %d, want 5", got)
	}
}
	`)
}

func TestGeneratedCode_RuntimeBehavior_FieldInitializers(t *testing.T) {
	src := `
public class FieldInitProgram {
    static int seed = 7;
    static int total = seed + 3;
    int left = 7;
    int right = left + 3;

    public FieldInitProgram() {}

    public int sum() {
        return left + right;
    }

    public static int run() {
        FieldInitProgram value = new FieldInitProgram();
        return value.sum() + total;
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected transpiler to produce non-empty Go output")
	}
	flat := strings.Join(strings.Fields(out), " ")

	if !strings.Contains(flat, "var ( seed int32 total int32 )") {
		t.Fatalf("expected all static fields to receive Java defaults before initialization, got:\n%s", out)
	}
	if !strings.Contains(flat, "func init() { __java2goExecution := stdjava.NewExecution() _ = __java2goExecution seed = 7 total = seed + 3 }") {
		t.Fatalf("expected source-ordered static field initialization after defaults, got:\n%s", out)
	}
	if !strings.Contains(out, "__java2goInitFields") {
		t.Fatalf("expected synthetic field initializer method in output, got:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestFieldInitializersBehavior(t *testing.T) {
	got := Run()
	if got != 27 {
		t.Fatalf("Run() = %d, want 27", got)
	}
}
	`)
}

func TestGeneratedCode_RuntimeBehavior_TryCatchFinally(t *testing.T) {
	src := `
public class RuntimeTryProgram {
    public static int run() {
        int total;
        total = 0;
        try {
            int denom;
            denom = 0;
            total += 10 / denom;
        } catch (Exception e) {
            total += 50;
        } finally {
            total += 3;
        }
        return total;
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected transpiler to produce non-empty Go output")
	}
	if !strings.Contains(out, "recover()") {
		t.Fatalf("expected try/catch lowering to include recover, got:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestTryCatchFinallyBehavior(t *testing.T) {
	got := Run()
	if got != 53 {
		t.Fatalf("Run() = %d, want 53", got)
	}
}
	`)
}

func TestGeneratedCode_RuntimeBehavior_TryFinallyReturnOverride(t *testing.T) {
	src := `
public class RuntimeTryFinallyProgram {
    public static int run() {
        try {
            return 10;
        } finally {
            return 20;
        }
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

func TestTryFinallyReturnOverrideBehavior(t *testing.T) {
	got := Run()
	if got != 20 {
		t.Fatalf("Run() = %d, want 20", got)
	}
}
	`)
}

func TestGeneratedCode_RuntimeBehavior_TryCatchFinallyReturnOverride(t *testing.T) {
	src := `
public class RuntimeTryCatchFinallyProgram {
    public static int run() {
        try {
            int denom;
            denom = 0;
            return 10 / denom;
        } catch (Exception e) {
            return 7;
        } finally {
            return 9;
        }
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

func TestTryCatchFinallyReturnOverrideBehavior(t *testing.T) {
	got := Run()
	if got != 9 {
		t.Fatalf("Run() = %d, want 9", got)
	}
}
`)
}

func TestGeneratedCode_RuntimeBehavior_TryCatchFinallyPanicOverride(t *testing.T) {
	src := `
public class RuntimeTryFinallyPanicProgram {
    public static int run() {
        try {
            return 1;
        } catch (Exception e) {
            return 2;
        } finally {
            int denom;
            denom = 0;
            return 3 / denom;
        }
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

func TestTryCatchFinallyPanicOverrideBehavior(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic from finally override, got none")
		}
	}()
	_ = Run()
}
	`)
}

func TestGeneratedCode_RuntimeBehavior_TryWithResources_CloseOrder(t *testing.T) {
	src := `
public class TraceHolder {
    String value;
    public TraceHolder() {
        this.value = "";
    }
    public void append(String piece) {
        this.value += piece;
    }
    public String get() {
        return this.value;
    }
}

public class RuntimeResource implements AutoCloseable {
    TraceHolder holder;
    String name;
    public RuntimeResource(TraceHolder holder, String name) {
        this.holder = holder;
        this.name = name;
    }
    public void close() {
        holder.append(name);
    }
}

public class RuntimeTryWithResourcesProgram {
    public static String run() {
        TraceHolder holder = new TraceHolder();
        try (RuntimeResource first = new RuntimeResource(holder, "A");
             RuntimeResource second = new RuntimeResource(holder, "B")) {
            holder.append("X");
        }
        return holder.get();
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

func TestTryWithResourcesCloseOrderBehavior(t *testing.T) {
	got := Run()
	if got != "XBA" {
		t.Fatalf("Run() = %q, want %q", got, "XBA")
	}
}
`)
}
