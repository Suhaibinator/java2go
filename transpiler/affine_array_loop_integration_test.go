package transpiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const affineLoopProgramSource = `
final class DenseGrid {
    private final double[] values;
    private final int size;

    DenseGrid(int size, int capacity) {
        this.size = size;
        this.values = new double[capacity];
    }

    DenseGrid(int size) {
        this.size = size;
        this.values = null;
    }

    double get(int row, int column) {
        return this.values[row * this.size + column];
    }

    void set(int row, int column, double value) {
        this.values[row * this.size + column] = value;
    }

    void add(int row, int column, double value) {
        this.values[row * this.size + column] =
            this.values[row * this.size + column] + value;
    }

    void addValueFirst(int row, int column, double value) {
        this.values[row * this.size + column] =
            value + this.values[row * this.size + column];
    }
}

public class AffineLoopProgram {
    public static DenseGrid grid(int size, int capacity) {
        return new DenseGrid(size, capacity);
    }

    public static DenseGrid pair(double first, double second) {
        DenseGrid grid = new DenseGrid(2, 2);
        grid.set(0, 0, first);
        grid.set(0, 1, second);
        return grid;
    }

    public static double run(DenseGrid grid, DenseGrid alias) {
        for (int row = 0; row < 2; row++) {
            for (int column = 0; column < 2; column++) {
                grid.set(row, column, row * 10.0 + column);
                grid.add(row, column, 0.5);
            }
        }
        for (int once = 0; once < 1; once++) {
            alias.set(1, 1, 25.0);
            grid.addValueFirst(1, 1, 2.0);
        }

        double total = 0.0;
        for (int row = 0; row < 2; row++) {
            for (int column = 0; column < 2; column++) {
                total += grid.get(row, column);
            }
        }
        return total;
    }

    public static double widened(DenseGrid grid) {
        short row = 1;
        byte column = 0;
        for (int once = 0; once < 1; once++) {
            grid.set(row, column, 3);
        }
        return grid.get(1, 0);
    }

    public static int overflow(DenseGrid grid) {
        for (int once = 0; once < 1; once++) {
            grid.set(65536, 0, 9.0);
        }
        return (int)grid.get(0, 0);
    }

    public static double zeroTrip(DenseGrid grid) {
        for (int once = 0; once < 0; once++) {
            return grid.get(0, 0);
        }
        return 7.0;
    }

    public static double oneTrip(DenseGrid grid) {
        for (int once = 0; once < 1; once++) {
            return grid.get(0, 0);
        }
        return -1.0;
    }

    public static double outOfBounds(DenseGrid grid, int row) {
        for (int once = 0; once < 1; once++) {
            return grid.get(row, 0);
        }
        return -1.0;
    }

    public static double nullBacking() {
        DenseGrid grid = new DenseGrid(1);
        for (int once = 0; once < 1; once++) {
            return grid.get(0, 0);
        }
        return -1.0;
    }

    public static double nestedRebind(DenseGrid current, DenseGrid replacement) {
        double total = 0.0;
        for (int outer = 0; outer < 1; outer++) {
            current = replacement;
            for (int inner = 0; inner < 2; inner++) {
                total += current.get(0, inner);
            }
        }
        return total;
    }

    public static double reassignedFallback(DenseGrid current, DenseGrid replacement) {
        for (int once = 0; once < 1; once++) {
            current = replacement;
            return current.get(0, 0);
        }
        return -1.0;
    }

    static int calls = 0;
    static int trace = 0;
    static int zero = 0;
    static int next() {
        calls++;
        return 0;
    }

    public static int effectfulFallback(DenseGrid grid) {
        calls = 0;
        for (int once = 0; once < 1; once++) {
            grid.set(next(), next(), 1.0);
        }
        return calls;
    }

    static int markIndex(int digit) {
        trace = trace * 10 + digit;
        return 0;
    }

    static double markValue(int digit) {
        trace = trace * 10 + digit;
        return 1.0;
    }

    public static int orderedEffectfulFallback(DenseGrid grid) {
        trace = 0;
        for (int once = 0; once < 1; once++) {
            grid.set(markIndex(1), markIndex(2), markValue(3));
        }
        return trace;
    }

    public static double panickingArgumentFallback(DenseGrid grid) {
        for (int once = 0; once < 1; once++) {
            return grid.get(1 / zero, 0);
        }
        return -1.0;
    }

    public static double labeledFallback(DenseGrid grid) {
        double total = 0.0;
        outer: for (int once = 0; once < 1; once++) {
            total += grid.get(0, 0);
            continue outer;
        }
        return total;
    }

    public static double nestedLabeledFallback(DenseGrid grid) {
        double total = 0.0;
        for (int outer = 0; outer < 1; outer++) {
            inner: for (int once = 0; once < 1; once++) {
                total = grid.get(0, 0);
                break inner;
            }
        }
        return total;
    }
}
`

func TestAffineArrayLoopFastPath_GeneratedShapeAndRuntime(t *testing.T) {
	out := renderGoFileFromJava(t, affineLoopProgramSource)
	flat := normalizeSpaces(out)
	for _, fragment := range []string{
		":= grid.Java2goAffineView0Values()",
		"stdjava.NewNullPointerException",
		"int(__java2goAffineArg0*",
		"+= __java2goAffineArg2",
		"__java2goAffineArg2 + __java2goAffine",
	} {
		if !strings.Contains(flat, fragment) {
			t.Fatalf("generated affine loop is missing %q:\n%s", fragment, out)
		}
	}
	for _, ordinaryCall := range []string{"grid.set(row, column", "grid.add(row, column", "grid.get(row, column", "alias.set(1, 1"} {
		if strings.Contains(flat, ordinaryCall) {
			t.Fatalf("hot-loop accessor call %q was not lowered:\n%s", ordinaryCall, out)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestAffineFastPathRuntime(t *testing.T) {
    grid := Grid(2, 4)
    if got := Run(grid, grid); got != 39.5 {
        t.Fatalf("Run() = %v, want 39.5", got)
    }
    if got := Widened(Grid(2, 4)); got != 3.0 {
        t.Fatalf("Widened() = %v, want 3", got)
    }
    if got := Overflow(Grid(65536, 1)); got != 9 {
        t.Fatalf("Overflow() = %v, want 9 (Java int32 index overflow)", got)
    }
    if got := NestedRebind(Pair(1, 2), Pair(3, 4)); got != 7 {
        t.Fatalf("NestedRebind() = %v, want 7", got)
    }
    if got := ReassignedFallback(Pair(1, 2), Pair(3, 4)); got != 3 {
        t.Fatalf("ReassignedFallback() = %v, want 3", got)
    }
    if got := EffectfulFallback(Grid(1, 1)); got != 2 {
        t.Fatalf("EffectfulFallback() evaluated args %d times, want 2", got)
    }
    if got := OrderedEffectfulFallback(Grid(1, 1)); got != 123 {
        t.Fatalf("OrderedEffectfulFallback() trace = %d, want 123", got)
    }
}
`)
}

func TestAffineArrayLoopFastPath_ExceptionFidelity(t *testing.T) {
	out := renderGoFileFromJava(t, affineLoopProgramSource)
	runGoTestInTempModule(t, out, `
package main

import (
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)

func recovered(call func()) (got interface{}) {
    defer func() { got = recover() }()
    call()
    return nil
}

func TestAffineExceptionFidelity(t *testing.T) {
    if got := ZeroTrip(nil); got != 7 {
        t.Fatalf("zero-trip nil receiver returned %v, want 7", got)
    }

    nullReceiver := stdjava.NormalizePanic(recovered(func() { OneTrip(nil) }))
    if !stdjava.CaughtAs(nullReceiver, "NullPointerException") || stdjava.CaughtAs(nullReceiver, "ArrayIndexOutOfBoundsException") {
        t.Fatalf("nil receiver normalized as %T (%v), want only NullPointerException", nullReceiver, nullReceiver)
    }

    nullArray := stdjava.NormalizePanic(recovered(func() { NullBacking() }))
    if !stdjava.CaughtAs(nullArray, "NullPointerException") {
        t.Fatalf("null backing array normalized as %T (%v), want NullPointerException", nullArray, nullArray)
    }

    for name, row := range map[string]int32{"negative": -1, "too-large": 1} {
        bounds := stdjava.NormalizePanic(recovered(func() { OutOfBounds(Grid(1, 1), row) }))
        if !stdjava.CaughtAs(bounds, "ArrayIndexOutOfBoundsException") || stdjava.CaughtAs(bounds, "NullPointerException") {
            t.Fatalf("%s index normalized as %T (%v), want only ArrayIndexOutOfBoundsException", name, bounds, bounds)
        }
    }

    arithmetic := stdjava.NormalizePanic(recovered(func() { PanickingArgumentFallback(nil) }))
    if !stdjava.CaughtAs(arithmetic, "ArithmeticException") || stdjava.CaughtAs(arithmetic, "NullPointerException") {
        t.Fatalf("panicking argument normalized as %T (%v), want ArithmeticException before receiver NPE", arithmetic, arithmetic)
    }
}
`)
}

func TestAffineArrayLoopFastPath_VersionsNonNullFastPathAndGuardedFallback(t *testing.T) {
	src := `
final class VersionGrid {
    private final double[] values;
    private final int size = 1;
    VersionGrid(double value) {
        this.values = new double[1];
        this.values[0] = value;
    }
    VersionGrid() { this.values = null; }
    double get(int row, int column) {
        return this.values[row * this.size + column];
    }
}
public class VersionedLoopProgram {
    public static VersionGrid allocated(double value) { return new VersionGrid(value); }
    public static VersionGrid nullBacked() { return new VersionGrid(); }
    public static double versioned(VersionGrid grid, boolean access, int trips) {
        double total = 7.0;
        for (int once = 0; once < trips; once++) {
            if (access) {
                total += grid.get(0, 0);
            }
        }
        return total;
    }
}
`
	out := renderGoFileFromJava(t, src)
	versioned := generatedFunctionText(out, "Versioned")
	if versioned == "" {
		t.Fatalf("generated Versioned function not found:\n%s", out)
	}
	flat := normalizeSpaces(versioned)
	elseIndex := strings.Index(flat, "} else {")
	if elseIndex < 0 {
		t.Fatalf("affine loop was not versioned:\n%s", versioned)
	}
	fast, guarded := flat[:elseIndex], flat[elseIndex:]
	for _, fragment := range []string{"grid != nil", "Values != nil", "for once := int32(0); once < trips; once++"} {
		if !strings.Contains(fast, fragment) {
			t.Fatalf("guard-free affine branch is missing %q:\n%s", fragment, versioned)
		}
	}
	if strings.Contains(fast, "NewNullPointerException") {
		t.Fatalf("non-null affine branch retained per-access null guards:\n%s", versioned)
	}
	if strings.Count(guarded, "NewNullPointerException") != 2 {
		t.Fatalf("invalid affine branch must retain receiver and backing-array guards:\n%s", versioned)
	}

	runGoTestInTempModule(t, out, `
package main

import (
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)

func versionedRecovered(call func()) (got interface{}) {
    defer func() { got = recover() }()
    call()
    return nil
}

func TestVersionedAffineRuntime(t *testing.T) {
    if got := Versioned(Allocated(3), true, 2); got != 13 {
        t.Fatalf("valid fast branch = %v, want 13", got)
    }
    if got := Versioned(nil, false, 1); got != 7 {
        t.Fatalf("conditional nil receiver = %v, want 7", got)
    }
    if got := Versioned(NullBacked(), false, 1); got != 7 {
        t.Fatalf("conditional null backing = %v, want 7", got)
    }
    if got := Versioned(nil, true, 0); got != 7 {
        t.Fatalf("zero-trip nil receiver = %v, want 7", got)
    }
    failures := map[string]func(){
        "receiver": func() { Versioned(nil, true, 1) },
        "backing": func() { Versioned(NullBacked(), true, 1) },
    }
    for name, call := range failures {
        recovered := stdjava.NormalizePanic(versionedRecovered(call))
        if !stdjava.CaughtAs(recovered, "NullPointerException") || stdjava.CaughtAs(recovered, "ArrayIndexOutOfBoundsException") {
            t.Fatalf("%s failure normalized as %T (%v), want only NullPointerException", name, recovered, recovered)
        }
    }
}
`)
}

func TestAffineArrayLoopFastPath_NestedVersionTracksNonNullPerBinding(t *testing.T) {
	src := `
final class MixedVersionGrid {
    private final int[] values;
    private final int size = 1;
    MixedVersionGrid(int value) {
        this.values = new int[1];
        this.values[0] = value;
    }
    MixedVersionGrid() { this.values = null; }
    int get(int row, int column) {
        return this.values[row * this.size + column];
    }
}
public class MixedVersionLoopProgram {
    public static MixedVersionGrid valid(int value) { return new MixedVersionGrid(value); }
    public static MixedVersionGrid nullBacked() { return new MixedVersionGrid(); }
    public static int run(MixedVersionGrid inherited, MixedVersionGrid changing,
                          MixedVersionGrid replacement) {
        int total = 0;
        for (int outer = 0; outer < 1; outer++) {
            changing = replacement;
            for (int inner = 0; inner < 1; inner++) {
                total += changing.get(0, 0);
                total += inherited.get(0, 0);
            }
        }
        return total;
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if strings.Count(flat, "Java2goAffineView0Values()") < 2 || !strings.Contains(flat, "NewNullPointerException") {
		t.Fatalf("mixed nested bindings did not produce nested versioning with guarded paths:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main

import (
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)

func mixedRecovered(call func()) (got interface{}) {
    defer func() { got = recover() }()
    call()
    return nil
}

func TestMixedBindingProofs(t *testing.T) {
    if got := Run(Valid(2), Valid(1), Valid(3)); got != 5 {
        t.Fatalf("valid mixed branches = %d, want 5", got)
    }
    failures := map[string]func(){
        "receiver": func() { Run(nil, Valid(1), Valid(3)) },
        "backing": func() { Run(NullBacked(), Valid(1), Valid(3)) },
    }
    for name, call := range failures {
        recovered := stdjava.NormalizePanic(mixedRecovered(call))
        if !stdjava.CaughtAs(recovered, "NullPointerException") || stdjava.CaughtAs(recovered, "ArrayIndexOutOfBoundsException") {
            t.Fatalf("inherited %s failure normalized as %T (%v), want only NullPointerException", name, recovered, recovered)
        }
    }
}
`)
}

func TestAffineArrayLoopFastPath_LabelElsewhereDisablesVersioning(t *testing.T) {
	src := `
final class LabeledBodyGrid {
    private final int[] values = new int[1];
    private final int size = 1;
    int get(int row, int column) { return this.values[row * this.size + column]; }
}
public class LabeledBodyLoopProgram {
    public static LabeledBodyGrid grid() { return new LabeledBodyGrid(); }
    public static int run(LabeledBodyGrid grid) {
        int total = 0;
        for (int outer = 0; outer < 1; outer++) {
            total += grid.get(0, 0);
            unrelated: for (int inner = 0; inner < 1; inner++) {
                total++;
                break unrelated;
            }
        }
        return total;
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if strings.Contains(flat, ":= grid.Java2goAffineView") || !strings.Contains(flat, "grid.getJava2goExecution(__java2goExecution, 0, 0)") {
		t.Fatalf("loop containing an unrelated label was duplicated instead of retaining ordinary dispatch:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestLabeledBodyFallback(t *testing.T) {
    if got := Run(Grid()); got != 1 {
        t.Fatalf("Run(Grid()) = %d, want 1", got)
    }
}
`)
}

func TestAffineArrayLoopFastPath_VersionedLoopReportsUnsupportedOnce(t *testing.T) {
	withCleanDiagnostics(t)
	src := `
final class DiagnosticGrid {
    private final int[] values = new int[1];
    private final int size = 1;
    int get(int row, int column) { return this.values[row * this.size + column]; }
}
public class DiagnosticVersionLoopProgram {
    public static int run(DiagnosticGrid grid) {
        int total = 0;
        for (int once = 0; once < 1; once++) {
            total += grid.get(0, 0);
            assert total >= 0;
        }
        return total;
    }
}
`
	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "Java2goAffineView") || strings.Count(out, "UNSUPPORTED") != 2 {
		t.Fatalf("versioned loop should retain an unsupported placeholder in both branches:\n%s", out)
	}
	diagnostics := collectedDiagnostics()
	if len(diagnostics) != 1 || diagnostics[0].Kind != "statement" || diagnostics[0].NodeType != "assert_statement" {
		t.Fatalf("versioned unsupported construct diagnostics = %#v, want one assert statement", diagnostics)
	}
}

func TestAffineArrayLoopFastPath_ConservativeFallbackShape(t *testing.T) {
	out := normalizeSpaces(renderGoFileFromJava(t, affineLoopProgramSource))
	for _, fragment := range []string{
		"current = replacement return current.getJava2goExecution(__java2goExecution, 0, 0)",
		"grid.setJava2goExecution(__java2goExecution, nextJava2goExecution(__java2goExecution), nextJava2goExecution(__java2goExecution), 1.0)",
		"grid.setJava2goExecution(__java2goExecution, markIndexJava2goExecution(__java2goExecution, 1), markIndexJava2goExecution(__java2goExecution, 2), markValueJava2goExecution(__java2goExecution, 3))",
		"return grid.getJava2goExecution(__java2goExecution, 1/func() int32 { AffineLoopProgramJava2goEnsureInitialized(__java2goExecution) return zero }(), 0)",
		"for once := int32(0); once < 1; once++ { func(dst *float64)",
		"(&total)(grid.getJava2goExecution(__java2goExecution, 0, 0)) continue __java2goLabel_",
		"for once := int32(0); once < 1; once++ { total = grid.getJava2goExecution(__java2goExecution, 0, 0) break __java2goLabel_",
	} {
		if !strings.Contains(out, fragment) {
			t.Fatalf("expected conservative fallback fragment %q:\n%s", fragment, out)
		}
	}
}

func TestAffineArrayLoopFastPath_ScopeAndHeaderFallback(t *testing.T) {
	src := `
interface GridReader { double read(); }
final class ScopedGrid {
    private final double[] values = new double[1];
    private final int size = 1;
    double get(int row, int column) { return this.values[row * this.size + column]; }
}
public class ScopedLoopProgram {
    static ScopedGrid current = new ScopedGrid();
    static int index = 0;

    public static double header(ScopedGrid grid) {
        double total = 0.0;
        for (int once = 0; once < grid.get(0, 0); once++) {
            total += 1.0;
        }
        return total;
    }

    public static double captured(ScopedGrid grid) {
        double total = 0.0;
        for (int once = 0; once < 1; once++) {
            GridReader reader = () -> grid.get(0, 0);
            total += reader.read();
        }
        return total;
    }

    public static double shadowed(ScopedGrid grid, ScopedGrid other) {
        double total = 0.0;
        for (int once = 0; once < 1; once++) {
            {
                ScopedGrid grid = other;
                total += grid.get(0, 0);
            }
        }
        return total;
    }

    public static double fieldReceiver() {
        double total = 0.0;
        for (int once = 0; once < 1; once++) {
            total += current.get(0, 0);
        }
        return total;
    }

    public static double fieldArgument(ScopedGrid grid) {
        for (int once = 0; once < 1; once++) {
            return grid.get(index, 0);
        }
        return -1.0;
    }

    public static <T extends ScopedGrid> double boundedReceiver(T grid) {
        for (int once = 0; once < 1; once++) {
            return grid.get(0, 0);
        }
        return -1.0;
    }
}
`
	out := normalizeSpaces(renderGoFileFromJava(t, src))
	if strings.Contains(out, ":= grid.Java2goAffineView") || strings.Contains(out, ":= current.Java2goAffineView") {
		t.Fatalf("header/captured/shadowed/field receiver unexpectedly received a loop cache:\n%s", out)
	}
	for _, ordinary := range []string{
		"grid.getJava2goExecution(__java2goExecution, 0, 0)",
		"grid.getJava2goExecution(__java2goExecution, func() int32 { ScopedLoopProgramJava2goEnsureInitialized(__java2goExecution) return index }(), 0)",
		"func() *scopedGrid { ScopedLoopProgramJava2goEnsureInitialized(__java2goExecution) return current }().getJava2goExecution(__java2goExecution, 0, 0)",
	} {
		if !strings.Contains(out, ordinary) {
			t.Fatalf("expected ordinary fallback call %q:\n%s", ordinary, out)
		}
	}
}

func TestAffineArrayLoopFastPath_NonFinalDispatchFallback(t *testing.T) {
	src := `
class BaseGrid {
    double get(int row, int column) { return 1.0; }
}
class ChildGrid extends BaseGrid {
    double get(int row, int column) { return 9.0; }
}
public class DispatchLoopProgram {
    public static BaseGrid child() { return new ChildGrid(); }
    public static double run(BaseGrid grid) {
        double total = 0.0;
        for (int once = 0; once < 1; once++) {
            total += grid.get(0, 0);
        }
        return total;
    }
}
`
	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "Java2goAffineView") {
		t.Fatalf("non-final dispatch unexpectedly received an affine view:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestDispatchFallback(t *testing.T) {
    if got := Run(Child()); got != 9 {
        t.Fatalf("Run(Child()) = %v, want dynamic override 9", got)
    }
}
`)
}

func TestAffineArrayLoopFastPath_GoKeywordReceiverCompiles(t *testing.T) {
	src := `
final class KeywordGrid {
    private final int[] values = new int[1];
    private final int size = 1;
    int get(int row, int column) { return this.values[row * this.size + column]; }
}
public class KeywordLoopProgram {
    public static int run() {
        KeywordGrid map = new KeywordGrid();
        for (int once = 0; once < 1; once++) {
            return map.get(0, 0);
        }
        return -1;
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "map_ := newKeywordGridJava2goExecution(__java2goExecution)") || !strings.Contains(flat, ":= map_.Java2goAffineView") {
		t.Fatalf("keyword receiver cache did not use its sanitized Go identifier:\n%s", out)
	}
	if strings.Contains(flat, ":= map.Java2goAffineView") {
		t.Fatalf("keyword receiver cache used the unsanitized Go keyword:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestKeywordReceiver(t *testing.T) {
    if got := Run(); got != 0 {
        t.Fatalf("Run() = %d, want 0", got)
    }
}
`)
}

func TestAffineArrayLoopFastPath_InjectedRuntimeNameCollisionFallback(t *testing.T) {
	src := `
final class CollisionGrid {
    private final int[] values = new int[1];
    private final int size = 1;
    int get(int row, int column) { return this.values[row * this.size + column]; }
}
public class CollisionLoopProgram {
    public static CollisionGrid grid() { return new CollisionGrid(); }
    public static int run(CollisionGrid grid, int panic, int nil, int stdjava) {
        for (int once = 0; once < 1; once++) {
            return grid.get(panic + nil + stdjava, 0);
        }
        return -1;
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if strings.Contains(flat, ":= grid.Java2goAffineView") || strings.Contains(flat, "NewNullPointerException") {
		t.Fatalf("shadowed injected runtime identifiers unexpectedly used the fast path:\n%s", out)
	}
	if !strings.Contains(flat, "return grid.getJava2goExecution(__java2goExecution, panic+nil+stdjava, 0)") {
		t.Fatalf("runtime-name collision did not retain ordinary dispatch:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestRuntimeNameCollisionFallback(t *testing.T) {
    if got := Run(Grid(), 0, 0, 0); got != 0 {
        t.Fatalf("Run(Grid(), 0, 0, 0) = %d, want 0", got)
    }
}
`)
}

func TestAffineArrayLoopFastPath_NestedSignatureTypeCollisionFallback(t *testing.T) {
	src := `
final class TypeShadowGrid {
    private final int[] values = new int[1];
    private final int size = 1;
    int get(int row, int column) { return this.values[row * this.size + column]; }
}
public class TypeShadowLoopProgram {
    public static TypeShadowGrid grid() { return new TypeShadowGrid(); }
    public static int receiverType(TypeShadowGrid typeShadowGrid) {
        for (int once = 0; once < 1; once++) {
            return typeShadowGrid.get(0, 0);
        }
        return -1;
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if strings.Contains(flat, "Java2goAffineView0Values() for") || strings.Contains(flat, "NewNullPointerException") {
		t.Fatalf("shadowed nested-signature type unexpectedly used the fast path:\n%s", out)
	}
	for _, ordinary := range []string{"typeShadowGrid.getJava2goExecution(__java2goExecution, 0, 0)"} {
		if !strings.Contains(flat, ordinary) {
			t.Fatalf("type-name collision did not retain ordinary call %q:\n%s", ordinary, out)
		}
	}
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestTypeCollisionFallback(t *testing.T) {
    grid := Grid()
    if got := ReceiverType(grid); got != 0 {
        t.Fatalf("ReceiverType(Grid()) = %d, want 0", got)
    }
}
`)
}

func TestAffineArrayLoopFastPath_NestedBinderRuntimeNameCollisionFallback(t *testing.T) {
	src := `
final class BinderGrid {
    private final int[] values = new int[1];
    private final int size = 1;
    int get(int row, int column) { return this.values[row * this.size + column]; }
}
public class BinderLoopProgram {
    public static BinderGrid grid() { return new BinderGrid(); }
    public static int run(BinderGrid grid, int[] source) {
        int result = 0;
        for (int outer = 0; outer < 1; outer++) {
            for (int nil : source) {
                result += grid.get(0, 0);
            }
        }
        return result;
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if strings.Contains(flat, "NewNullPointerException") || strings.Contains(flat, ":= grid.Java2goAffineView") {
		t.Fatalf("nested binder shadowing nil unexpectedly used the fast path:\n%s", out)
	}
	if !strings.Contains(flat, "grid.getJava2goExecution(__java2goExecution, 0, 0)") {
		t.Fatalf("nested binder collision did not retain ordinary dispatch:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main
import (
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)
func TestNestedBinderFallback(t *testing.T) {
    source := stdjava.PrimitiveArrayLiteral[int32](stdjava.PrimitiveTypeID("int"), 1)
    if got := Run(Grid(), source); got != 0 {
        t.Fatalf("Run(Grid(), int[]{1}) = %d, want 0", got)
    }
}
`)
}

func TestAffineArrayLoopFastPath_CrossPackageAliasBinderCollisionFallback(t *testing.T) {
	sourceRoot := t.TempDir()
	writeJavaTestSource(t, sourceRoot, "affinecross/p/Grid.java", `
package affinecross.p;
public final class Grid {
    private final int[] values = new int[1];
    private final int size = 1;
    public int get(int row, int column) { return this.values[row * this.size + column]; }
}
`)
	writeJavaTestSource(t, sourceRoot, "affinecross/app/App.java", `
package affinecross.app;
import affinecross.p.Grid;
public class App {
    public static Grid grid() { return new Grid(); }
    public static int run(Grid grid, int[] source) {
        int result = 0;
        for (int outer = 0; outer < 1; outer++) {
            for (int p : source) {
                result += grid.get(0, 0);
            }
        }
        return result;
    }
}
`)

	outputs := convertJavaProjectDir(t, sourceRoot)
	appOut := outputs["affinecross/app/App.go"]
	flat := normalizeSpaces(appOut)
	if strings.Contains(flat, "NewNullPointerException") || strings.Contains(flat, ":= grid.Java2goAffineView") {
		t.Fatalf("cross-package alias shadowed by nested binder unexpectedly used the fast path:\n%s", appOut)
	}
	if !strings.Contains(flat, "for _, p := range stdjava.PrimitiveArrayIterationElements(source)") || !strings.Contains(flat, "grid.GetJava2goExecution(__java2goExecution, 0, 0)") {
		t.Fatalf("cross-package alias collision did not retain ordinary dispatch:\n%s", appOut)
	}

	moduleRoot := t.TempDir()
	goMod := "module affinecross\n\ngo 1.25.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => " + repoRootDir(t) + "\n"
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for relative, generated := range outputs {
		relative = strings.TrimPrefix(filepath.ToSlash(relative), "affinecross/")
		path := filepath.Join(moduleRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create generated package: %v", err)
		}
		if err := os.WriteFile(path, []byte(generated), 0o644); err != nil {
			t.Fatalf("write generated source: %v", err)
		}
	}
	testPath := filepath.Join(moduleRoot, "app", "affine_alias_test.go")
	if err := os.WriteFile(testPath, []byte(`package app
import (
    "testing"
    "github.com/NickyBoy89/java2go/stdjava"
)
func TestAliasCollisionFallback(t *testing.T) {
    source := stdjava.PrimitiveArrayLiteral[int32](stdjava.PrimitiveTypeID("int"), 1)
    if got := Run(Grid(), source); got != 0 {
        t.Fatalf("Run(Grid(), int[]{1}) = %d, want 0", got)
    }
}
`), 0o644); err != nil {
		t.Fatalf("write runtime test: %v", err)
	}
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = moduleRoot
	if output, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("tidy cross-package alias collision module:\n%s", output)
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = moduleRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("cross-package alias collision generated code failed:\n%s", output)
	}
}

func TestAffineArrayLoopFastPath_TypeParameterCollisionFallback(t *testing.T) {
	src := `
final class TypeParameterGrid {
    private final int[] values = new int[1];
    private final int size = 1;
    int get(int row, int column) { return this.values[row * this.size + column]; }
}
final class TypeParameterDoubleGrid {
    private final double[] values = new double[1];
    private final int size = 1;
    void set(int row, int column, double value) {
        this.values[row * this.size + column] = value;
    }
}
public class TypeParameterLoopProgram<panic> {
    public static TypeParameterGrid grid() { return new TypeParameterGrid(); }
    public static TypeParameterDoubleGrid doubleGrid() { return new TypeParameterDoubleGrid(); }
    public static TypeParameterLoopProgram<Integer> program() {
        return new TypeParameterLoopProgram<Integer>();
    }
    public int classRuntime(TypeParameterGrid grid, panic ignored) {
        for (int once = 0; once < 1; once++) {
            return grid.get(0, 0);
        }
        return -1;
    }
    public static <stdjava> int methodRuntime(TypeParameterGrid grid, stdjava ignored) {
        for (int once = 0; once < 1; once++) {
            return grid.get(0, 0);
        }
        return -1;
    }
    public static <float64> void methodType(TypeParameterDoubleGrid grid, float64 ignored) {
        for (int once = 0; once < 1; once++) {
            grid.set(0, 0, 1.0);
            break;
        }
    }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if strings.Contains(flat, "NewNullPointerException") || strings.Contains(flat, ":= grid.Java2goAffineView") {
		t.Fatalf("type-parameter identifier collision unexpectedly used the fast path:\n%s", out)
	}
	if strings.Count(flat, "grid.getJava2goExecution(__java2goExecution, 0, 0)") != 2 ||
		!strings.Contains(flat, "grid.setJava2goExecution(__java2goExecution, 0, 0, 1.0)") {
		t.Fatalf("type-parameter collisions did not retain all ordinary calls:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestTypeParameterFallback(t *testing.T) {
    grid := Grid()
    if got := Program().ClassRuntime(grid, int32(0)); got != 0 {
        t.Fatalf("ClassRuntime() = %d, want 0", got)
    }
    if got := MethodRuntime(grid, int32(0)); got != 0 {
        t.Fatalf("MethodRuntime() = %d, want 0", got)
    }
    MethodType(DoubleGrid(), int32(0))
}
`)
}

func TestAffineArrayLoopFastPath_SyntheticStaticInitializerFallback(t *testing.T) {
	src := `
final class StaticInitGrid {
    private final int[] values = new int[1];
    private final int size = 1;
    int get(int row, int column) { return this.values[row * this.size + column]; }
}
public class StaticInitLoopProgram {
    static int result = -1;
    static {
        StaticInitGrid grid = new StaticInitGrid();
        for (int once = 0; once < 1; once++) {
            int panic = 0;
            result = panic + grid.get(0, 0);
        }
    }
    public static int result() { return result; }
}
`
	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if strings.Contains(flat, "NewNullPointerException") || strings.Contains(flat, ":= grid.Java2goAffineView") {
		t.Fatalf("synthetic static-initializer scope unexpectedly used the fast path:\n%s", out)
	}
	if !strings.Contains(flat, "grid.getJava2goExecution(__java2goExecution, 0, 0)") {
		t.Fatalf("static initializer did not retain ordinary dispatch:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestStaticInitializerFallback(t *testing.T) {
    if got := Result(); got != 0 {
        t.Fatalf("Result() = %d, want 0", got)
    }
}
`)
}

func TestAffineArrayLoopFastPath_CacheNameDoesNotShadowReceiverType(t *testing.T) {
	const loopStart = 2048
	className := fmt.Sprintf("__java2goAffine%dValues", loopStart)
	prefix := fmt.Sprintf(`
final class %[1]s {
    private final int[] values = new int[1];
    private final int size = 1;
    int get(int row, int column) { return this.values[row * this.size + column]; }
}
public class CacheNameLoopProgram {
    public static %[1]s makeGrid() { return new %[1]s(); }
    public static int run(%[1]s grid) {
        `, className)
	loop := `for (int once = 0; once < 1; once++) {
            return grid.get(0, 0);
        }
        return -1;
    }
}
`
	if len(prefix) >= loopStart {
		t.Fatalf("test prefix length %d must remain below fixed loop start %d", len(prefix), loopStart)
	}
	src := prefix + strings.Repeat(" ", loopStart-len(prefix)) + loop
	if got := strings.Index(src, "for (int once"); got != loopStart {
		t.Fatalf("constructed loop start = %d, want %d", got, loopStart)
	}

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	arrayName := fmt.Sprintf("__java2goAffine%dValues", loopStart)
	strideName := fmt.Sprintf("__java2goAffine%dStride", loopStart)
	if !strings.Contains(flat, arrayName+"0, "+strideName+" := grid.Java2goAffineView") {
		t.Fatalf("cache name was not disambiguated from synthesized receiver type:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main
import "testing"
func TestCacheTypeNameCollision(t *testing.T) {
    if got := Run(MakeGrid()); got != 0 {
        t.Fatalf("Run(MakeGrid()) = %d, want 0", got)
    }
}
`)
}
