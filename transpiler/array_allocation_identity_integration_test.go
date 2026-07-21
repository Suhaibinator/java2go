package transpiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestJavaArrayAllocationsUseIdentityPreservingRuntime(t *testing.T) {
	src := `
public class ArrayAllocationIdentityProgram {
	private static int[] fieldEmpty = new int[] {};

	public static int[] fieldEmpty() {
		return fieldEmpty;
	}

	public static int[] returnedEmpty() {
		return new int[] {};
	}

	public static int[] inferredEmpty() {
		var local = new int[] {};
		return local;
	}

	public static int[][] partialRank() {
		return new int[2][];
	}

	public static int[][] nestedEmpty() {
		return new int[][] {{}, {}};
	}

	public static <T> int genericLength(T[] values) {
		return values.length;
	}

    public static int shapes() {
        int[] sizedEmpty = new int[0];
        int[] literalEmpty = new int[] {};
        int[][] rows = new int[2][0];
        int[][] literals = new int[][] {{}, {7}};
        return sizedEmpty.length * 10000
                + literalEmpty.length * 1000
                + rows.length * 100
                + rows[0].length * 10
                + literals[1][0];
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "make([]int32") || strings.Contains(out, "make([][]int32") {
		t.Fatalf("generated Java arrays bypassed the identity-preserving allocator:\n%s", out)
	}
	if got := strings.Count(out, "stdjava.NewArray["); got < 3 {
		t.Fatalf("sized and multidimensional rows must use NewArray; got %d calls:\n%s", got, out)
	}
	if got := strings.Count(out, "stdjava.ArrayLiteral["); got < 4 {
		t.Fatalf("outer and nested literals must use ArrayLiteral; got %d calls:\n%s", got, out)
	}

	runGeneratedWithStdjava(t, out, `
package main

import (
    "testing"
    "time"

    "github.com/NickyBoy89/java2go/stdjava"
)

func requireIndependentArrayMonitors(t *testing.T, first, second []int32) {
    t.Helper()
    firstMonitor := stdjava.MonitorEnter(first)
    acquired := make(chan struct{})
    go func() {
        secondMonitor := stdjava.MonitorEnter(second)
        close(acquired)
        stdjava.MonitorExit(secondMonitor)
    }()

    select {
    case <-acquired:
        stdjava.MonitorExit(firstMonitor)
    case <-time.After(2 * time.Second):
        stdjava.MonitorExit(firstMonitor)
        t.Fatal("distinct generated zero-length arrays shared one monitor")
    }
}

func TestArrayAllocationShapes(t *testing.T) {
    if got := Shapes(); got != 207 {
        t.Fatalf("Shapes() = %d, want 207", got)
    }

    fieldFirst := FieldEmpty()
    fieldAlias := FieldEmpty()
    if len(fieldFirst) != 0 || cap(fieldFirst) != 1 || &fieldFirst[:1][0] != &fieldAlias[:1][0] {
        t.Fatalf("field array did not retain one zero-length Java object: first=%v alias=%v", fieldFirst, fieldAlias)
    }

    returnedFirst := ReturnedEmpty()
    returnedSecond := ReturnedEmpty()
    inferred := InferredEmpty()
    for name, value := range map[string][]int32{"returned first": returnedFirst, "returned second": returnedSecond, "inferred": inferred} {
        if len(value) != 0 || cap(value) != 1 {
            t.Fatalf("%s shape = len %d cap %d, want len 0 cap 1", name, len(value), cap(value))
        }
    }
    requireIndependentArrayMonitors(t, returnedFirst, returnedSecond)
    requireIndependentArrayMonitors(t, returnedFirst, inferred)

    partial := PartialRank()
    if len(partial) != 2 || partial[0] != nil || partial[1] != nil {
        t.Fatalf("partial array = %#v, want two null rows", partial)
    }

    nested := NestedEmpty()
    if len(nested) != 2 || len(nested[0]) != 0 || cap(nested[0]) != 1 || len(nested[1]) != 0 || cap(nested[1]) != 1 {
        t.Fatalf("nested empty array has wrong shape: %#v", nested)
    }
    requireIndependentArrayMonitors(t, nested[0], nested[1])

    if got := GenericLength[string](stdjava.NewArray[string](0)); got != 0 {
        t.Fatalf("GenericLength(empty) = %d, want 0", got)
    }
}
`)
}

func TestJavaArrayAllocationPreservesDimensionEvaluationAndNegativeOrdering(t *testing.T) {
	src := `
public class ArrayAllocationOrderProgram {
    private static int trace;

    private static int dimension(int marker, int value) {
        trace = trace * 10 + marker;
        return value;
    }

    public static int order() {
        trace = 0;
        try {
            int[][] ignored = new int[dimension(1, 0)][dimension(2, -1)];
            trace = trace * 10 + 9;
        } catch (NegativeArraySizeException expected) {
            trace = trace * 10 + 8;
        }
        return trace;
	    }
	}
`

	out := renderGoFileFromJava(t, src)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestArrayAllocationOrder(t *testing.T) {
    if got := Order(); got != 128 {
        t.Fatalf("Order() = %d, want 128", got)
    }
}
	`)
}

func TestJavaArrayRuntimeImportAliasesAvoidPackagesLocalsAndTypeParameters(t *testing.T) {
	sourceRoot := t.TempDir()
	writeJavaTestSource(t, sourceRoot, "collision/stdjava/Element.java", `
package collision.stdjava;
public class Element {}
`)
	writeJavaTestSource(t, sourceRoot, "collision/app/AliasApplication.java", `
package collision.app;
import collision.stdjava.Element;

public class AliasApplication {
    public static Element[] allocate(int stdjava) {
        return new Element[stdjava];
    }

    public static <stdjava> int generic(stdjava ignored) {
        int[] values = new int[0];
        return values.length;
    }

    public static int run() {
        return allocate(2).length + generic("ignored");
    }
}
`)

	outputs := convertJavaProjectDir(t, sourceRoot)
	appOutput := outputs["collision/app/AliasApplication.go"]
	if !strings.Contains(appOutput, `"collision/stdjava"`) || !strings.Contains(appOutput, `"github.com/NickyBoy89/java2go/stdjava"`) {
		t.Fatalf("generated application must retain both user and runtime stdjava imports:\n%s", appOutput)
	}

	moduleRoot := t.TempDir()
	goMod := "module collision\n\ngo 1.26.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => " + repoRootDir(t) + "\n"
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write alias collision go.mod: %v", err)
	}
	for relative, generated := range outputs {
		relative = strings.TrimPrefix(filepath.ToSlash(relative), "collision/")
		path := filepath.Join(moduleRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create generated alias collision package: %v", err)
		}
		if err := os.WriteFile(path, []byte(generated), 0o644); err != nil {
			t.Fatalf("write generated alias collision source: %v", err)
		}
	}
	testPath := filepath.Join(moduleRoot, "app", "alias_application_test.go")
	if err := os.WriteFile(testPath, []byte(`package app
import "testing"
func TestAliasApplication(t *testing.T) {
    if got := Run(); got != 2 { t.Fatalf("Run() = %d, want 2", got) }
}
`), 0o644); err != nil {
		t.Fatalf("write alias collision runtime test: %v", err)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = moduleRoot
	if output, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("tidy alias collision module:\n%s", output)
	}
	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = moduleRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("alias collision generated code failed:\n%s", output)
	}
}
