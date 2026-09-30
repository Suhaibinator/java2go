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

func TestAffineArrayRowLoop_IntermediateVersionDoesNotDuplicateInheritedHoists(t *testing.T) {
	src := `
final class DuplicateHoistGrid {
    private final int[] values = new int[16];
    private final int stride = 16;
    DuplicateHoistGrid(int seed) {
        for (int index = 0; index < 16; index++) {
            values[index] = seed + index;
        }
    }
    int get(int row, int column) { return this.values[row * this.stride + column]; }
    void set(int row, int column, int value) { this.values[row * this.stride + column] = value; }
}
public class DuplicateHoistProgram {
    public static int run(DuplicateHoistGrid source, DuplicateHoistGrid target) {
        for (int outer = 0; outer < 1; outer++) {
            target = source;
            for (int middle = 0; middle < 1; middle++) {
                for (int column = 0; column < 16; column++) {
                    int first = source.get(0, column);
                    int second = source.get(0, column);
                    int current = source.get(0, column);
                    source.set(0, column, current + first + second);
                }
                int ignored = target.get(0, 0);
            }
        }
        return target.get(0, 0) + target.get(0, 15);
    }
}
`
	out := renderGoFileFromJava(t, src)
	run := generatedFunctionText(out, "Run")
	if !strings.Contains(run, "__java2goAffineRow") {
		t.Fatalf("intermediate version lost the inherited-binding row specialization:\n%s", run)
	}
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestIntermediateVersionRuntime(t *testing.T) {
    if got := Run(newDuplicateHoistGrid(1), newDuplicateHoistGrid(100)); got != 51 {
        t.Fatalf("Run() = %d, want 51", got)
    }
}
`)
}

func TestAffineArrayRowLoop_RowPreambleDoesNotEscapeCommonProof(t *testing.T) {
	src := `
final class DependentHoistGrid {
    private final int[] values = new int[32];
    private final int stride = 32;
    DependentHoistGrid(int seed) {
        for (int index = 0; index < 32; index++) {
            values[index] = seed + index;
        }
    }
    int get(int row, int column) { return this.values[row * this.stride + column]; }
    void set(int row, int column, int value) { this.values[row * this.stride + column] = value; }
}
public class DependentHoistProgram {
    public static int run(DependentHoistGrid grid) {
        for (int owner = 0; owner < 1; owner++) {
            for (int segment = 0; segment < 2; segment++) {
                int start = segment * 16;
                int end = start + 16;
                for (int column = start; column < end; column++) {
                    int current = grid.get(0, column);
                    grid.set(0, column, current + 1);
                }
            }
        }
        return grid.get(0, 0) + grid.get(0, 31);
    }
}
`
	out := renderGoFileFromJava(t, src)
	run := generatedFunctionText(out, "Run")
	if !strings.Contains(run, "__java2goAffineRow") {
		t.Fatalf("dependent row preamble fixture lost row specialization:\n%s", run)
	}
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestDependentHoistRuntime(t *testing.T) {
    if got := Run(newDependentHoistGrid(1)); got != 35 {
        t.Fatalf("Run() = %d, want 35", got)
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
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte("module rowcollision\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => "+repoRootDir(t)+"\n"), 0o644); err != nil {
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

func writeAffineRowLocalHelper(t *testing.T, sourceRoot string) {
	writeJavaTestSource(t, sourceRoot, "rowlocals/Helper.java", `
package rowlocals;
final class Helper {
    static int sum(int len, int int64) { return len + int64; }
}
`)
}

const affineWholeRangeProgramSource = `
final class WholeRangeGrid {
    private final int[] values;
    private final int stride;

    WholeRangeGrid(int stride, int capacity, int seed) {
        this.stride = stride;
        this.values = new int[capacity];
        for (int index = 0; index < capacity; index++) {
            this.values[index] = seed + index;
        }
    }

    int get(int row, int column) {
        return this.values[row * this.stride + column];
    }

    void set(int row, int column, int value) {
        this.values[row * this.stride + column] = value;
    }
}

public class AffineWholeRangeProgram {
    public static WholeRangeGrid grid(int stride, int capacity, int seed) {
        return new WholeRangeGrid(stride, capacity, seed);
    }

    public static int blocked(WholeRangeGrid source, WholeRangeGrid target,
                              int extent, int step) {
        for (int rowBlock = 0; rowBlock < extent; rowBlock += step) {
            int rowLimit = Math.min(rowBlock + step, extent);
            for (int depthBlock = 0; depthBlock < extent; depthBlock += step) {
                int depthLimit = Math.min(depthBlock + step, extent);
                for (int columnBlock = 0; columnBlock < extent; columnBlock += step) {
                    int columnLimit = Math.min(columnBlock + step, extent);
                    for (int row = rowBlock; row < rowLimit; row++) {
                        for (int depth = depthBlock; depth < depthLimit; depth++) {
                            for (int column = columnBlock; column < columnLimit; column++) {
                                int value = source.get(depth, column);
                                int current = target.get(row, column);
                                target.set(row, column, current + value);
                            }
                        }
                    }
                }
            }
        }
        return extent == 0 ? 7 : target.get(0, 0) + target.get(extent - 1, extent - 1);
    }

    public static int mutatedLimit(WholeRangeGrid source, WholeRangeGrid target,
                                   int extent, int step) {
        for (int rowBlock = 0; rowBlock < extent; rowBlock += step) {
            int rowLimit = Math.min(rowBlock + step, extent);
            rowLimit = rowLimit - 1;
            for (int depthBlock = 0; depthBlock < extent; depthBlock += step) {
                int depthLimit = Math.min(depthBlock + step, extent);
                for (int columnBlock = 0; columnBlock < extent; columnBlock += step) {
                    int columnLimit = Math.min(columnBlock + step, extent);
                    for (int row = rowBlock; row < rowLimit; row++) {
                        for (int depth = depthBlock; depth < depthLimit; depth++) {
                            for (int column = columnBlock; column < columnLimit; column++) {
                                int value = source.get(depth, column);
                                int current = target.get(row, column);
                                target.set(row, column, current + value);
                            }
                        }
                    }
                }
            }
        }
        return 0;
    }

    public static int mismatchedStep(WholeRangeGrid source, WholeRangeGrid target,
                                     int extent, int step, int otherStep) {
        for (int rowBlock = 0; rowBlock < extent; rowBlock += step) {
            int rowLimit = Math.min(rowBlock + step, extent);
            for (int depthBlock = 0; depthBlock < extent; depthBlock += otherStep) {
                int depthLimit = Math.min(depthBlock + otherStep, extent);
                for (int columnBlock = 0; columnBlock < extent; columnBlock += step) {
                    int columnLimit = Math.min(columnBlock + step, extent);
                    for (int row = rowBlock; row < rowLimit; row++) {
                        for (int depth = depthBlock; depth < depthLimit; depth++) {
                            for (int column = columnBlock; column < columnLimit; column++) {
                                int value = source.get(depth, column);
                                int current = target.get(row, column);
                                target.set(row, column, current + value);
                            }
                        }
                    }
                }
            }
        }
        return 0;
    }
}
`

func TestAffineArrayRowLoop_WholeRangeHoistingShapeAndRuntime(t *testing.T) {
	out := renderGoFileFromJava(t, affineWholeRangeProgramSource)
	blocked := generatedFunctionText(out, "Blocked")
	flat := normalizeSpaces(blocked)
	for _, fragment := range []string{
		"WholeProduct64",
		"LastRow64",
		"Extent > 0",
		"Step > 0",
		"for rowBlock := int32(0)",
		"Slice0 :=",
		"Slice1 :=",
	} {
		if !strings.Contains(flat, fragment) {
			t.Fatalf("whole-range affine lowering is missing %q:\n%s", fragment, blocked)
		}
	}
	wholeProof := strings.Index(blocked, "WholeProduct64")
	ownerLoop := strings.Index(blocked, "for rowBlock :=")
	resultSlice := strings.Index(blocked, "Slice1 :=")
	rowLoop := strings.Index(blocked, "for row :=")
	rightSlice := strings.Index(blocked, "Slice0 :=")
	depthLoop := strings.Index(blocked, "for depth :=")
	if wholeProof < 0 || ownerLoop < 0 || wholeProof > ownerLoop || resultSlice < rowLoop || rightSlice < depthLoop {
		t.Fatalf("whole-range/result-row/right-row preambles were not hoisted to their expected scopes:\n%s", blocked)
	}
	if strings.Contains(generatedFunctionText(out, "MutatedLimit"), "WholeProduct64") {
		t.Fatalf("mutated block limit unexpectedly received a whole-range proof:\n%s", generatedFunctionText(out, "MutatedLimit"))
	}
	if strings.Contains(generatedFunctionText(out, "MismatchedStep"), "WholeProduct64") {
		t.Fatalf("mismatched block steps unexpectedly received a whole-range proof:\n%s", generatedFunctionText(out, "MismatchedStep"))
	}
	mutatedCounterSource := strings.Replace(affineWholeRangeProgramSource,
		"int rowLimit = Math.min(rowBlock + step, extent);",
		"int rowLimit = Math.min(rowBlock + step, extent); rowBlock = rowBlock;", 1)
	mutatedCounter := renderGoFileFromJava(t, mutatedCounterSource)
	if strings.Contains(generatedFunctionText(mutatedCounter, "Blocked"), "WholeProduct64") {
		t.Fatalf("a mutated block counter unexpectedly received a whole-range proof:\n%s", generatedFunctionText(mutatedCounter, "Blocked"))
	}
	shadowedSource := strings.Replace(affineWholeRangeProgramSource,
		"public class AffineWholeRangeProgram {",
		"final class Math { static int min(int a, int b) { return a < b ? a : b; } }\npublic class AffineWholeRangeProgram {", 1)
	shadowed := renderGoFileFromJava(t, shadowedSource)
	if strings.Contains(generatedFunctionText(shadowed, "Blocked"), "WholeProduct64") {
		t.Fatalf("a user-defined Math.min unexpectedly received the java.lang.Math whole-range proof:\n%s", generatedFunctionText(shadowed, "Blocked"))
	}
	inheritedMathSource := strings.Replace(affineWholeRangeProgramSource,
		"public class AffineWholeRangeProgram {",
		`final class NonstandardMath {
    int min(int a, int b) { return a < b ? a : b; }
}
class WholeRangeBase {
    protected static final NonstandardMath Math = new NonstandardMath();
}
public class AffineWholeRangeProgram extends WholeRangeBase {`, 1)
	inheritedMath := renderGoFileFromJava(t, inheritedMathSource)
	if strings.Contains(generatedFunctionText(inheritedMath, "Blocked"), "WholeProduct64") {
		t.Fatalf("an inherited field named Math unexpectedly received the java.lang.Math whole-range proof:\n%s", generatedFunctionText(inheritedMath, "Blocked"))
	}
	ambiguousLimitSource := strings.Replace(affineWholeRangeProgramSource,
		"for (int rowBlock = 0; rowBlock < extent; rowBlock += step) {",
		`{
            int columnLimit = 0;
            if (columnLimit < 0) return columnLimit;
        }
        for (int rowBlock = 0; rowBlock < extent; rowBlock += step) {`, 1)
	ambiguousLimit := renderGoFileFromJava(t, ambiguousLimitSource)
	if strings.Contains(generatedFunctionText(ambiguousLimit, "Blocked"), "WholeProduct64") {
		t.Fatalf("an ambiguous disjoint block-limit name unexpectedly received a whole-range proof:\n%s", generatedFunctionText(ambiguousLimit, "Blocked"))
	}

	runGoTestInTempModule(t, out, `
package main

import (
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)

func wholeRangeRecovered(call func()) (got interface{}) {
    defer func() { got = recover() }()
    call()
    return nil
}

func wholeRangeReference(source, target *wholeRangeGrid, extent, step int32) int32 {
    for rowBlock := int32(0); rowBlock < extent; rowBlock += step {
        rowLimit := rowBlock + step
        if extent < rowLimit { rowLimit = extent }
        for depthBlock := int32(0); depthBlock < extent; depthBlock += step {
            depthLimit := depthBlock + step
            if extent < depthLimit { depthLimit = extent }
            for columnBlock := int32(0); columnBlock < extent; columnBlock += step {
                columnLimit := columnBlock + step
                if extent < columnLimit { columnLimit = extent }
                for row := rowBlock; row < rowLimit; row++ {
                    for depth := depthBlock; depth < depthLimit; depth++ {
                        for column := columnBlock; column < columnLimit; column++ {
                            value := source.get(depth, column)
                            current := target.get(row, column)
                            target.set(row, column, current + value)
                        }
                    }
                }
            }
        }
    }
    if extent == 0 { return 7 }
    return target.get(0, 0) + target.get(extent - 1, extent - 1)
}

func TestAffineWholeRangeRuntime(t *testing.T) {
    if got := Blocked(newWholeRangeGrid(16, 256, 1), newWholeRangeGrid(16, 256, 0), 16, 16); got != 4367 {
        t.Fatalf("Blocked(valid) = %d, want 4367", got)
    }
    if got := Blocked(newWholeRangeGrid(16, 256, 1), newWholeRangeGrid(16, 256, 0), 16, 2147483647); got != 4367 {
        t.Fatalf("Blocked(step-overflow fallback) = %d, want 4367", got)
    }
    if got := Blocked(nil, nil, 0, 16); got != 7 {
        t.Fatalf("Blocked(zero trip, nil) = %d, want 7", got)
    }

    actualAlias := newWholeRangeGrid(16, 256, 1)
    referenceAlias := newWholeRangeGrid(16, 256, 1)
    gotAlias := Blocked(actualAlias, actualAlias, 16, 16)
    wantAlias := wholeRangeReference(referenceAlias, referenceAlias, 16, 16)
    if gotAlias != wantAlias {
        t.Fatalf("Blocked(alias) = %d, reference %d", gotAlias, wantAlias)
    }
    actualValues := actualAlias.values.Elements
    referenceValues := referenceAlias.values.Elements
    for index := range actualValues {
        if actualValues[index] != referenceValues[index] {
            t.Fatalf("Blocked(alias) values[%d] = %d, reference %d", index, actualValues[index], referenceValues[index])
        }
    }

    recovered := stdjava.NormalizePanic(wholeRangeRecovered(func() {
        Blocked(newWholeRangeGrid(16, 255, 1), newWholeRangeGrid(16, 256, 0), 16, 16)
    }))
    if !stdjava.CaughtAs(recovered, "ArrayIndexOutOfBoundsException") || stdjava.CaughtAs(recovered, "NullPointerException") {
        t.Fatalf("short backing normalized as %T (%v), want only ArrayIndexOutOfBoundsException", recovered, recovered)
    }

    overflowed := stdjava.NormalizePanic(wholeRangeRecovered(func() {
        Blocked(newWholeRangeGrid(1073741824, 256, 1), newWholeRangeGrid(16, 256, 0), 16, 16)
    }))
    if !stdjava.CaughtAs(overflowed, "ArrayIndexOutOfBoundsException") || stdjava.CaughtAs(overflowed, "NullPointerException") {
        t.Fatalf("overflowing stride normalized as %T (%v), want only ArrayIndexOutOfBoundsException", overflowed, overflowed)
    }

    nullReceiver := stdjava.NormalizePanic(wholeRangeRecovered(func() {
        Blocked(nil, newWholeRangeGrid(16, 256, 0), 16, 16)
    }))
    if !stdjava.CaughtAs(nullReceiver, "NullPointerException") || stdjava.CaughtAs(nullReceiver, "ArrayIndexOutOfBoundsException") {
        t.Fatalf("null receiver normalized as %T (%v), want only NullPointerException", nullReceiver, nullReceiver)
    }
}
`)
}

func TestAffineArrayRowLoop_SiblingMethodLocalsDoNotDisableSpecialization(t *testing.T) {
	sourceRoot := t.TempDir()
	writeAffineRowLocalHelper(t, sourceRoot)
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
	start := strings.Index(out, "func "+name+"Java2goExecution(")
	if start < 0 {
		start = strings.Index(out, "func "+name+"(")
	}
	if start < 0 {
		return ""
	}
	function := out[start:]
	if next := strings.Index(function[1:], "\nfunc "); next >= 0 {
		function = function[:next+1]
	}
	return function
}
