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
	if strings.Contains(out, "stdjava.NewArray[") || strings.Contains(out, "stdjava.ArrayLiteral[") {
		t.Fatalf("generated Java arrays used the legacy slice-only ABI:\n%s", out)
	}
	for _, expected := range []string{
		`stdjava.PrimitiveArrayLiteral[int32](stdjava.PrimitiveIntTypeID)`,
		`stdjava.NewPrimitiveArray[int32](0, stdjava.PrimitiveIntTypeID)`,
		`stdjava.NewMultiArrayOf[int32](stdjava.PrimitiveIntTypeID, 2, int32(2), int32(0))`,
		`stdjava.ReferenceArrayLiteralOf[*stdjava.PrimitiveArray[int32]](stdjava.ArrayTypeID(stdjava.PrimitiveIntTypeID)`,
		`stdjava.ReferenceArrayLength(values)`,
	} {
		if !strings.Contains(out, expected) {
			t.Fatalf("generated arrays must retain the exact Java component descriptor in %q:\n%s", expected, out)
		}
	}

	runGeneratedWithStdjava(t, out, `
package main

import (
    "testing"
    "time"

    "github.com/NickyBoy89/java2go/stdjava"
)

func requireIndependentArrayMonitors(t *testing.T, first, second any) {
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
	intType := stdjava.PrimitiveTypeID("int")
	intArrayType := stdjava.ArrayTypeID(intType)
	if fieldFirst == nil || fieldFirst != fieldAlias || len(fieldFirst.Elements) != 0 {
        t.Fatalf("field array did not retain one zero-length Java object: first=%v alias=%v", fieldFirst, fieldAlias)
    }
	if fieldFirst.ComponentType() != intType || fieldFirst.JavaArrayTypeID() != intArrayType {
		t.Fatalf("field int[] descriptors = component %q array %q, want %q and %q", fieldFirst.ComponentType(), fieldFirst.JavaArrayTypeID(), intType, intArrayType)
	}

    returnedFirst := ReturnedEmpty()
    returnedSecond := ReturnedEmpty()
    inferred := InferredEmpty()
	for name, value := range map[string]*stdjava.PrimitiveArray[int32]{"returned first": returnedFirst, "returned second": returnedSecond, "inferred": inferred} {
		if value == nil || len(value.Elements) != 0 {
			t.Fatalf("%s shape = %#v, want a non-nil zero-length Java array", name, value)
        }
		if value.ComponentType() != intType || value.JavaArrayTypeID() != intArrayType {
			t.Fatalf("%s descriptors = component %q array %q, want %q and %q", name, value.ComponentType(), value.JavaArrayTypeID(), intType, intArrayType)
		}
    }
	if returnedFirst == returnedSecond || returnedFirst == inferred || returnedSecond == inferred {
		t.Fatal("distinct zero-length allocations collapsed to the same Java array object")
	}
    requireIndependentArrayMonitors(t, returnedFirst, returnedSecond)
    requireIndependentArrayMonitors(t, returnedFirst, inferred)

    partial := PartialRank()
	if stdjava.ReferenceArrayLength(partial) != 2 || partial.ComponentType() != intArrayType || partial.JavaArrayTypeID() != stdjava.ArrayTypeID(intArrayType) {
		t.Fatalf("partial array descriptors/length are wrong: component=%q array=%q length=%d", partial.ComponentType(), partial.JavaArrayTypeID(), stdjava.ReferenceArrayLength(partial))
	}
	if stdjava.ReferenceArrayGet[*stdjava.PrimitiveArray[int32]](partial, 0, intArrayType) != nil || stdjava.ReferenceArrayGet[*stdjava.PrimitiveArray[int32]](partial, 1, intArrayType) != nil {
        t.Fatalf("partial array = %#v, want two null rows", partial)
    }

    nested := NestedEmpty()
	if stdjava.ReferenceArrayLength(nested) != 2 || nested.ComponentType() != intArrayType {
		t.Fatalf("nested int[][] descriptors/length are wrong: component=%q length=%d", nested.ComponentType(), stdjava.ReferenceArrayLength(nested))
	}
	nestedFirst := stdjava.ReferenceArrayGet[*stdjava.PrimitiveArray[int32]](nested, 0, intArrayType)
	nestedSecond := stdjava.ReferenceArrayGet[*stdjava.PrimitiveArray[int32]](nested, 1, intArrayType)
	if nestedFirst == nil || nestedSecond == nil || len(nestedFirst.Elements) != 0 || len(nestedSecond.Elements) != 0 {
        t.Fatalf("nested empty array has wrong shape: %#v", nested)
    }
	if nestedFirst == nestedSecond {
		t.Fatal("two nested empty literals collapsed to the same Java array object")
	}
	requireIndependentArrayMonitors(t, nestedFirst, nestedSecond)

	genericEmpty := stdjava.NewReferenceArrayOf[string](0, stdjava.StringTypeID)
	if genericEmpty.ComponentType() != stdjava.StringTypeID || genericEmpty.JavaArrayTypeID() != stdjava.ArrayTypeID(stdjava.StringTypeID) {
		t.Fatalf("String[] descriptors = component %q array %q", genericEmpty.ComponentType(), genericEmpty.JavaArrayTypeID())
	}
	if got := GenericLength[string](genericEmpty); got != 0 {
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
