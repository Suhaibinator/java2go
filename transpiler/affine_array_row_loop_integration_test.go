package transpiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const affineRowLoopProgramSource = `
final class RowSliceGrid {
    private final int[] values;
    private final int stride;

    RowSliceGrid(int stride, int capacity, int seed) {
        this.stride = stride;
        this.values = new int[capacity];
        for (int index = 0; index < capacity; index++) {
            this.values[index] = seed + index;
        }
    }

    RowSliceGrid(int stride) {
        this.stride = stride;
        this.values = null;
    }

    int get(int row, int column) {
        return this.values[row * this.stride + column];
    }

    void set(int row, int column, int value) {
        this.values[row * this.stride + column] = value;
    }

    void add(int row, int column, int value) {
        this.values[row * this.stride + column] =
                this.values[row * this.stride + column] + value;
    }
}

public class AffineRowLoopProgram {
    public static RowSliceGrid grid(int stride, int capacity, int seed) {
        return new RowSliceGrid(stride, capacity, seed);
    }

    public static RowSliceGrid nullBacked(int stride) {
        return new RowSliceGrid(stride);
    }

    public static int run(RowSliceGrid source, RowSliceGrid target,
                          int row, int start, int end) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = start; column < end; column++) {
                int value = source.get(row, column);
                int current = target.get(row, column);
                target.set(row, column, current + value);
            }
        }
        return target.get(row, start) + target.get(row, end - 1);
    }

    public static int wrappedProduct(RowSliceGrid source, RowSliceGrid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                int value = source.get(65536, column);
                target.set(0, column, value);
            }
        }
        return target.get(0, 0) + target.get(0, 15);
    }

    public static int additionOverflow(RowSliceGrid source, RowSliceGrid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 1; column < 17; column++) {
                int value = source.get(2147483647, column);
                target.set(0, column, value);
            }
        }
        return 0;
    }

    public static int invalidBounds(RowSliceGrid source, RowSliceGrid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                int value = source.get(1, column);
                target.set(0, column, value);
            }
        }
        return 0;
    }

    public static int conditional(RowSliceGrid source, RowSliceGrid target,
                                  boolean access) {
        int result = 7;
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                if (access) {
                    int value = source.get(1, column);
                    result += value;
                    target.set(0, column, result);
                }
            }
        }
        return result;
    }

    public static int zeroTrip(RowSliceGrid source, RowSliceGrid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 16; column < 16; column++) {
                int value = source.get(1, column);
                target.set(0, column, value);
            }
        }
        return 11;
    }

    public static int controlFlow(RowSliceGrid source, RowSliceGrid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                if (column == 1) {
                    continue;
                }
                int value = source.get(0, column);
                int current = target.get(0, column);
                target.set(0, column, current + value);
                if (column == 2) {
                    break;
                }
            }
        }
        return target.get(0, 0) + target.get(0, 1)
                + target.get(0, 2) + target.get(0, 3);
    }

    public static int alias(RowSliceGrid grid) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                int current = grid.get(0, column);
                grid.set(0, column, current + current);
            }
        }
        return grid.get(0, 0) + grid.get(0, 15);
    }
}
`

func TestAffineArrayRowLoop_GeneratedShapeAndRuntime(t *testing.T) {
	out := renderGoFileFromJava(t, affineRowLoopProgramSource)
	run := generatedFunctionText(out, "Run")
	flat := normalizeSpaces(run)
	for _, fragment := range []string{
		"Span64 := int64(",
		"Span64 >= 16",
		"Product64",
		"int64(",
		"< int64(len(",
		"Slice0 :=",
		"Slice1 :=",
		"Slice0[__java2goAffineRow",
		"Slice1[__java2goAffineRow",
		"for __java2goAffineRow",
		":= range __java2goAffineRow",
		"column := __java2goAffineRow",
	} {
		if !strings.Contains(flat, fragment) {
			t.Fatalf("row-slice fast path is missing %q:\n%s", fragment, run)
		}
	}
	if strings.Contains(flat, "_ = __java2goAffineRow") {
		t.Fatalf("row-slice hot loop contains a synthetic unused-value discard:\n%s", run)
	}
	if strings.Count(flat, "Values[int(__java2goAffineArg0*") < 2 {
		t.Fatalf("row-slice bounds failure did not retain the flat affine fallback:\n%s", run)
	}
	if !strings.Contains(flat, "func(__java2goAffineReceiver *rowSliceGrid, __java2goAffineArg0 int32, __java2goAffineArg1 int32)") {
		t.Fatalf("row-slice access removed typed receiver/argument staging:\n%s", run)
	}
	if controlFlow := generatedFunctionText(out, "ControlFlow"); strings.Contains(controlFlow, "__java2goAffineRow") {
		t.Fatalf("conditional control flow unexpectedly received row-slice specialization:\n%s", controlFlow)
	}

	runGoTestInTempModule(t, out, `
package main

import (
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)

func rowRecovered(call func()) (got interface{}) {
    defer func() { got = recover() }()
    call()
    return nil
}

func TestAffineRowRuntime(t *testing.T) {
    if got := Run(Grid(16, 32, 10), Grid(16, 32, 0), 1, 0, 16); got != 114 {
        t.Fatalf("Run() = %d, want 114", got)
    }
    if got := Run(Grid(16, 16, 10), Grid(16, 16, 0), 0, 0, 4); got != 26 {
        t.Fatalf("Run(short span) = %d, want 26", got)
    }
    // 65536*65536 wraps to zero in Java int arithmetic. Product overflow must
    // choose the flat fallback, which retains that result.
    if got := WrappedProduct(Grid(65536, 16, 5), Grid(16, 16, 0)); got != 25 {
        t.Fatalf("WrappedProduct() = %d, want 25", got)
    }
    if got := Conditional(Grid(16, 16, 1), Grid(16, 16, 0), false); got != 7 {
        t.Fatalf("Conditional(invalid, false) = %d, want 7", got)
    }
    if got := ZeroTrip(Grid(16, 16, 1), Grid(16, 16, 0)); got != 11 {
        t.Fatalf("ZeroTrip(invalid) = %d, want 11", got)
    }
    if got := ZeroTrip(nil, nil); got != 11 {
        t.Fatalf("ZeroTrip(nil, nil) = %d, want 11", got)
    }
    if got := ControlFlow(Grid(16, 16, 10), Grid(16, 16, 0)); got != 28 {
        t.Fatalf("ControlFlow() = %d, want 28", got)
    }
    if got := Alias(Grid(16, 16, 1)); got != 34 {
        t.Fatalf("Alias() = %d, want 34", got)
    }

    failures := map[string]func(){
        "addition-overflow": func() { AdditionOverflow(Grid(1, 32, 1), Grid(32, 32, 0)) },
        "invalid-bounds": func() { InvalidBounds(Grid(16, 16, 1), Grid(16, 16, 0)) },
    }
    for name, call := range failures {
        recovered := stdjava.NormalizePanic(rowRecovered(call))
        if !stdjava.CaughtAs(recovered, "ArrayIndexOutOfBoundsException") || stdjava.CaughtAs(recovered, "NullPointerException") {
            t.Fatalf("%s normalized as %T (%v), want only ArrayIndexOutOfBoundsException", name, recovered, recovered)
        }
    }

    nullBacking := stdjava.NormalizePanic(rowRecovered(func() {
        Run(NullBacked(16), Grid(16, 16, 0), 0, 0, 16)
    }))
    if !stdjava.CaughtAs(nullBacking, "NullPointerException") || stdjava.CaughtAs(nullBacking, "ArrayIndexOutOfBoundsException") {
        t.Fatalf("null backing normalized as %T (%v), want only NullPointerException", nullBacking, nullBacking)
    }
}
`)
}

func TestAffineArrayRowLoop_UnstableAndNonCanonicalFallback(t *testing.T) {
	src := `
final class RowFallbackGrid {
    private final int[] values = new int[16];
    private final int stride = 16;
    int get(int row, int column) { return this.values[row * this.stride + column]; }
    void set(int row, int column, int value) { this.values[row * this.stride + column] = value; }
}
public class RowFallbackProgram {
    public static int unstableEnd(RowFallbackGrid source, RowFallbackGrid target,
                                  int row, int end) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < end; column++) {
                int value = source.get(row, column);
                target.set(row, column, value);
                end--;
            }
        }
        return end;
    }
    public static int unstableRow(RowFallbackGrid source, RowFallbackGrid target, int row) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                int value = source.get(row, column);
                target.set(row, column, value);
                row++;
            }
        }
        return row;
    }
    public static int unstableCounter(RowFallbackGrid source, RowFallbackGrid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                int value = source.get(0, column);
                target.set(0, column, value);
                column++;
            }
        }
        return 0;
    }
    public static int prefixUpdate(RowFallbackGrid source, RowFallbackGrid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; ++column) {
                int value = source.get(0, column);
                target.set(0, column, value);
            }
        }
        return 0;
    }
    public static int nonUnitUpdate(RowFallbackGrid source, RowFallbackGrid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column += 1) {
                int value = source.get(0, column);
                target.set(0, column, value);
            }
        }
        return 0;
    }
    public static int inclusive(RowFallbackGrid source, RowFallbackGrid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column <= 15; column++) {
                int value = source.get(0, column);
                target.set(0, column, value);
            }
        }
        return 0;
    }
    public static int nested(RowFallbackGrid source, RowFallbackGrid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                int value = source.get(0, column);
                target.set(0, column, value);
                for (int nested = 0; nested < 1; nested++) {
                    int nestedValue = source.get(0, column);
                    target.set(0, column, nestedValue);
                }
            }
        }
        return 0;
    }
    public static int localNames(RowFallbackGrid source, RowFallbackGrid target,
                                 int len, int int64) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                int value = source.get(0, column);
                target.set(0, column, value);
            }
        }
        return len + int64;
    }
    public static int multipleUpdates(RowFallbackGrid source, RowFallbackGrid target) {
        int ticks = 0;
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++, ticks++) {
                int value = source.get(0, column);
                target.set(0, column, value);
            }
        }
        return ticks;
    }
}
`
	out := renderGoFileFromJava(t, src)
	for _, functionName := range []string{
		"UnstableEnd", "UnstableRow", "UnstableCounter", "PrefixUpdate",
		"NonUnitUpdate", "Inclusive", "Nested", "LocalNames",
		"MultipleUpdates",
	} {
		function := generatedFunctionText(out, functionName)
		if strings.Contains(function, "__java2goAffineRow") {
			t.Fatalf("%s unexpectedly received row-slice specialization:\n%s", functionName, function)
		}
		if !strings.Contains(function, "Java2goAffineView") {
			t.Fatalf("%s lost the existing affine null-versioning path:\n%s", functionName, function)
		}
	}
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestRowFallbacksCompile(t *testing.T) {
    if got := LocalNames(newRowFallbackGrid(), newRowFallbackGrid(), 2, 3); got != 5 {
        t.Fatalf("LocalNames() = %d, want 5", got)
    }
}
`)
}

func TestAffineArrayRowLoop_ConditionalExpressionGating(t *testing.T) {
	src := `
package rowexpr;
final class RowExpressionGrid {
    private final int[] values = new int[16];
    private final int stride = 16;
    int get(int row, int column) { return this.values[row * this.stride + column]; }
    void set(int row, int column, int value) { this.values[row * this.stride + column] = value; }
}
public class RowExpressionProgram {
    public static int unrelatedTernary(RowExpressionGrid source, RowExpressionGrid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                int neighbor = column == 0 ? 15 : column - 1;
                int value = source.get(0, column);
                target.set(0, column, value + neighbor - neighbor);
            }
        }
        return target.get(0, 15);
    }
    public static int selectedTernary(RowExpressionGrid source, RowExpressionGrid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                int value = column == 0
                        ? source.get(0, column)
                        : target.get(0, column);
                target.set(0, column, value);
            }
        }
        return target.get(0, 15);
    }
    public static int shortCircuit(RowExpressionGrid source, RowExpressionGrid target) {
        int count = 0;
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                boolean active = source.get(0, column) > 0
                        && target.get(0, column) > 0;
                count += active ? 1 : 0;
            }
        }
        return count;
    }
}
`
	out := renderGoFileFromJava(t, src)
	if function := generatedFunctionText(out, "UnrelatedTernary"); !strings.Contains(function, "__java2goAffineRow") {
		t.Fatalf("unrelated ternary suppressed the straight-line row specialization:\n%s", function)
	}
	for _, functionName := range []string{"SelectedTernary", "ShortCircuit"} {
		function := generatedFunctionText(out, functionName)
		if strings.Contains(function, "__java2goAffineRow") {
			t.Fatalf("%s conditionally evaluates a selected call but received row specialization:\n%s", functionName, function)
		}
		if !strings.Contains(function, "Java2goAffineView") {
			t.Fatalf("%s lost ordinary affine versioning:\n%s", functionName, function)
		}
	}
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestConditionalExpressionRuntime(t *testing.T) {
    if got := UnrelatedTernary(newRowExpressionGrid(), newRowExpressionGrid()); got != 0 {
        t.Fatalf("UnrelatedTernary() = %d, want 0", got)
    }
    if got := SelectedTernary(newRowExpressionGrid(), newRowExpressionGrid()); got != 0 {
        t.Fatalf("SelectedTernary() = %d, want 0", got)
    }
    if got := ShortCircuit(newRowExpressionGrid(), newRowExpressionGrid()); got != 0 {
        t.Fatalf("ShortCircuit() = %d, want 0", got)
    }
}
`)
}

func TestAffineArrayRowLoop_SiblingPackageIdentifiersDisableSpecialization(t *testing.T) {
	sourceRoot := t.TempDir()
	writeJavaTestSource(t, sourceRoot, "rowcollision/len.java", `
package rowcollision;
final class len {}
`)
	writeJavaTestSource(t, sourceRoot, "rowcollision/int64.java", `
package rowcollision;
final class int64 {}
`)
	writeJavaTestSource(t, sourceRoot, "rowcollision/Grid.java", `
package rowcollision;
public final class Grid {
    private final int[] values = new int[16];
    private final int stride = 16;
    public int get(int row, int column) { return this.values[row * this.stride + column]; }
    public void set(int row, int column, int value) { this.values[row * this.stride + column] = value; }
}
`)
	writeJavaTestSource(t, sourceRoot, "rowcollision/App.java", `
package rowcollision;
public class App {
    public static int run(Grid source, Grid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                int value = source.get(0, column);
                target.set(0, column, value);
            }
        }
        return target.get(0, 0);
    }
}
`)

	outputs := convertJavaProjectDir(t, sourceRoot)
	appOut := outputs["rowcollision/App.go"]
	if strings.Contains(appOut, "__java2goAffineRow") {
		t.Fatalf("sibling package identifiers len/int64 did not disable row specialization:\n%s", appOut)
	}
	if !strings.Contains(appOut, "Java2goAffineView") {
		t.Fatalf("sibling identifier fallback lost ordinary affine versioning:\n%s", appOut)
	}

	moduleRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte("module rowcollision\n\ngo 1.25.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => "+repoRootDir(t)+"\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for relative, generated := range outputs {
		relative = strings.TrimPrefix(filepath.ToSlash(relative), "rowcollision/")
		path := filepath.Join(moduleRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create generated package directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(generated), 0o644); err != nil {
			t.Fatalf("write generated source: %v", err)
		}
	}
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = moduleRoot
	if output, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("tidy sibling identifier fallback module:\n%s", output)
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = moduleRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("sibling identifier fallback generated code failed:\n%s", output)
	}
}

func TestAffineArrayRowLoop_SiblingMethodLocalsDoNotDisableSpecialization(t *testing.T) {
	sourceRoot := t.TempDir()
	writeJavaTestSource(t, sourceRoot, "rowlocals/Helper.java", `
package rowlocals;
final class Helper {
    static int sum(int len, int int64) { return len + int64; }
}
`)
	writeJavaTestSource(t, sourceRoot, "rowlocals/Grid.java", `
package rowlocals;
public final class Grid {
    private final int[] values = new int[16];
    private final int stride = 16;
    public int get(int row, int column) { return this.values[row * this.stride + column]; }
    public void set(int row, int column, int value) { this.values[row * this.stride + column] = value; }
}
`)
	writeJavaTestSource(t, sourceRoot, "rowlocals/App.java", `
package rowlocals;
public class App {
    public static int run(Grid source, Grid target) {
        for (int outer = 0; outer < 1; outer++) {
            for (int column = 0; column < 16; column++) {
                int value = source.get(0, column);
                target.set(0, column, value);
            }
        }
        return target.get(0, 15);
    }
}
`)

	outputs := convertJavaProjectDir(t, sourceRoot)
	appOut := outputs["rowlocals/App.go"]
	if !strings.Contains(appOut, "__java2goAffineRow") {
		t.Fatalf("sibling method locals len/int64 suppressed row specialization:\n%s", appOut)
	}
}

func TestAffineArrayLoopUsedNames_ReservesTypeIdentifiers(t *testing.T) {
	helper := setupParseHelper(t, `
class CollisionProgram {
    void run() {
        for (int column = 0; column < 16; column++) {
            __java2goAffineRow123ColumnStart marker = null;
        }
    }
}
class __java2goAffineRow123ColumnStart {}
`)
	loop := findNode(helper.File.Ast, "for_statement")
	if loop == nil {
		t.Fatal("collision fixture has no for statement")
	}
	used := affineLoopUsedNames(loop, helper.File.Source, helper.Ctx)
	if _, ok := used["__java2goAffineRow123ColumnStart"]; !ok {
		t.Fatalf("type identifier was not reserved: %v", used)
	}
}

func generatedFunctionText(out, name string) string {
	start := strings.Index(out, "func "+name+"(")
	if start < 0 {
		return ""
	}
	function := out[start:]
	if next := strings.Index(function[1:], "\nfunc "); next >= 0 {
		function = function[:next+1]
	}
	return function
}
