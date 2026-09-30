package transpiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMultidimensionalArrayDimensionsEvaluateOnceBeforeChecks(t *testing.T) {
	src := `
final class __java2goDimension0 {}
final class __java2goArray {}
final class __java2goIndex0 {}
final class __java2goIndex1 {}
final class arr {}
final class ind {}
final class ind0 {}

public class MultidimensionalArrayEvaluationProgram {
    private static int trace;
    private static int firstCalls;
    private static int secondCalls;
    private static int thirdCalls;

    private static int dimension(int marker, int value) {
        trace = trace * 10 + marker;
        if (marker == 1) firstCalls++;
        if (marker == 2) secondCalls++;
        if (marker == 3) thirdCalls++;
        return value;
    }

    private static int throwingDimension() {
        trace = trace * 10 + 2;
        throw new IllegalStateException("dimension");
    }

    public static int negativeOrder() {
        trace = 0;
        try {
            int[][][] ignored = new int[dimension(1, 2)][dimension(2, -1)][dimension(3, 4)];
            trace = trace * 10 + 9;
        } catch (NegativeArraySizeException expected) {
            trace = trace * 10 + 8;
        } catch (RuntimeException wrong) {
            trace = trace * 10 + 7;
        }
        return trace;
    }

    public static int singleNegative() {
        trace = 0;
        try {
            int[] ignored = new int[dimension(1, -2)];
            trace = trace * 10 + 9;
        } catch (NegativeArraySizeException expected) {
            trace = trace * 10 + 8;
        } catch (RuntimeException wrong) {
            trace = trace * 10 + 7;
        }
        return trace;
    }

    public static int zeroBeforeNegative() {
        trace = 0;
        try {
            int[][][] ignored = new int[dimension(1, 0)][dimension(2, -1)][dimension(3, 4)];
            trace = trace * 10 + 9;
        } catch (NegativeArraySizeException expected) {
            trace = trace * 10 + 8;
        }
        return trace;
    }

    public static int laterExceptionWins() {
        trace = 0;
        try {
            int[][][] ignored = new int[dimension(1, -1)][throwingDimension()][dimension(3, 4)];
            trace = trace * 10 + 9;
        } catch (IllegalStateException expected) {
            trace = trace * 10 + 5;
        } catch (NegativeArraySizeException wrong) {
            trace = trace * 10 + 8;
        }
        return trace;
    }

    public static int evaluateOnceAndShape() {
        firstCalls = 0;
        secondCalls = 0;
        thirdCalls = 0;
        int[][][] values = new int[dimension(1, 2)][dimension(2, 3)][dimension(3, 4)];
        return firstCalls * 100000 + secondCalls * 10000 + thirdCalls * 1000
                + values.length * 100 + values[0].length * 10 + values[0][0].length;
    }

    public static int promotedDimensions() {
        byte first = 2;
        short second = 3;
        char third = 4;
        int[][][] values = new int[first][second][third];
        return values.length * 100 + values[0].length * 10 + values[0][0].length;
    }

    public static int promotedNarrowOperations() {
        byte byteMin = -128;
        byte byteHigh = 64;
        short shortMin = -32768;
        char charHigh = (char) 32768;
        int[] unaryByte = new int[-byteMin];
        int[] shiftedByte = new int[byteHigh << 2];
        int[] unaryShort = new int[-shortMin];
        int[] shiftedChar = new int[charHigh << 1];
        return unaryByte.length + shiftedByte.length
                + unaryShort.length + shiftedChar.length;
    }

    public static int partialRank() {
        int[][][] twoExplicit = new int[2][3][];
        int[][][] oneExplicit = new int[4][][];
        return twoExplicit.length * 1000
                + twoExplicit[0].length * 100
                + (twoExplicit[0][0] == null ? 10 : 0)
                + (oneExplicit[0] == null ? 1 : 0);
    }

    public static int binderCollisions() {
        __java2goDimension0[][] dimensions = new __java2goDimension0[2][1];
        arr[][] arrays = new arr[2][2];
        ind[][][] indexes = new ind[1][2][3];
        ind0[][][] nestedIndexes = new ind0[3][2][1];
        __java2goArray[][] generatedArrays = new __java2goArray[1][4];
        __java2goIndex0[][][] generatedIndexes = new __java2goIndex0[1][1][5];
        __java2goIndex1[][][][] generatedNestedIndexes = new __java2goIndex1[1][1][1][6];
        return dimensions.length * 1000 + dimensions[0].length * 100
                + arrays.length * 10 + arrays[0].length
                + indexes[0][0].length + nestedIndexes[0].length
                + generatedArrays[0].length + generatedIndexes[0][0].length
                + generatedNestedIndexes[0][0][0].length;
    }

    public static <__java2goDimension0, __java2goArray, __java2goIndex0, __java2goIndex1, arr, ind, ind0> int typeParameterBindersCompile() {
        int[][] values = new int[2][3];
        return values.length + values[0].length;
    }
}
`

	out := renderGoFileFromJava(t, src)
	function := generatedFunctionText(out, "NegativeOrder")
	if strings.Count(function, "dimensionJava2goExecution(") != 3 {
		t.Fatalf("each source dimension call must occur exactly once in generated code:\n%s", function)
	}
	allocation := `stdjava.NewMultiArrayOf[int32](stdjava.PrimitiveIntTypeID, 3,`
	if !strings.Contains(function, allocation) {
		t.Fatalf("multidimensional int array must retain its primitive descriptor and total rank in the checked allocator:\n%s", function)
	}
	firstDimension := strings.Index(function, "dimensionJava2goExecution(__java2goExecution, 1, 2)")
	secondDimension := strings.Index(function, "dimensionJava2goExecution(__java2goExecution, 2, -1)")
	thirdDimension := strings.Index(function, "dimensionJava2goExecution(__java2goExecution, 3, 4)")
	if firstDimension < 0 || secondDimension <= firstDimension || thirdDimension <= secondDimension {
		t.Fatalf("dimension expressions must be passed once in Java left-to-right order before runtime checks:\n%s", function)
	}
	typeParameterStart := strings.Index(out, "func TypeParameterBindersCompileJava2goExecution")
	if typeParameterStart < 0 {
		t.Fatalf("missing generated generic collision method:\n%s", out)
	}
	typeParameterTail := out[typeParameterStart:]
	typeParameterEnd := strings.Index(typeParameterTail, "\n}")
	if typeParameterEnd < 0 {
		t.Fatalf("could not isolate generated generic collision method:\n%s", typeParameterTail)
	}
	typeParameterFunction := typeParameterTail[:typeParameterEnd+2]
	if !strings.Contains(typeParameterFunction, `stdjava.NewMultiArrayOf[int32](stdjava.PrimitiveIntTypeID, 2, int32(2), int32(3))`) {
		t.Fatalf("generic type-parameter collisions must retain the descriptor-based multidimensional allocator:\n%s", typeParameterFunction)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestMultidimensionalArrayEvaluationRuntime(t *testing.T) {
    cases := []struct {
        name string
        got int32
        want int32
    }{
        {"negative-order", NegativeOrder(), 1238},
        {"single-negative", SingleNegative(), 18},
        {"zero-before-negative", ZeroBeforeNegative(), 1238},
        {"later-exception-wins", LaterExceptionWins(), 125},
        {"once-and-shape", EvaluateOnceAndShape(), 111234},
        {"promoted-byte-short-char", PromotedDimensions(), 234},
        {"promoted-narrow-operations", PromotedNarrowOperations(), 98688},
        {"partial-rank", PartialRank(), 2311},
        {"binder-collisions", BinderCollisions(), 2142},
    }
    for _, tc := range cases {
        if tc.got != tc.want {
            t.Errorf("%s = %d, want %d", tc.name, tc.got, tc.want)
        }
    }
}
`)
}

func TestMultidimensionalArrayBindersAvoidImportAliases(t *testing.T) {
	sourceRoot := t.TempDir()
	writeJavaTestSource(t, sourceRoot, "collision/arr/ArrElement.java", `
package collision.arr;
public final class ArrElement {}
`)
	writeJavaTestSource(t, sourceRoot, "collision/ind/IndElement.java", `
package collision.ind;
public final class IndElement {}
`)
	writeJavaTestSource(t, sourceRoot, "collision/ind0/Ind0Element.java", `
package collision.ind0;
public final class Ind0Element {}
`)
	writeJavaTestSource(t, sourceRoot, "collision/__java2goDimension0/DimensionElement.java", `
package collision.__java2goDimension0;
public final class DimensionElement {}
`)
	writeJavaTestSource(t, sourceRoot, "collision/__java2goArray/ArrayElement.java", `
package collision.__java2goArray;
public final class ArrayElement {}
`)
	writeJavaTestSource(t, sourceRoot, "collision/__java2goIndex0/IndexElement.java", `
package collision.__java2goIndex0;
public final class IndexElement {}
`)
	writeJavaTestSource(t, sourceRoot, "collision/__java2goIndex1/Index1Element.java", `
package collision.__java2goIndex1;
public final class Index1Element {}
`)
	writeJavaTestSource(t, sourceRoot, "collision/app/Application.java", `
package collision.app;
import collision.arr.ArrElement;
import collision.ind.IndElement;
import collision.ind0.Ind0Element;
import collision.__java2goDimension0.DimensionElement;
import collision.__java2goArray.ArrayElement;
import collision.__java2goIndex0.IndexElement;
import collision.__java2goIndex1.Index1Element;
public final class Application {
    public static int run() {
        ArrElement[][] arrValues = new ArrElement[2][3];
        IndElement[][][] indValues = new IndElement[1][2][4];
        Ind0Element[][][] ind0Values = new Ind0Element[3][2][1];
        DimensionElement[][] dimensionValues = new DimensionElement[4][5];
        ArrayElement[][] generatedArrayValues = new ArrayElement[1][6];
        IndexElement[][][] generatedIndexValues = new IndexElement[1][1][7];
        Index1Element[][][][] generatedIndex1Values = new Index1Element[1][1][1][8];
        return arrValues.length * 10000 + arrValues[0].length * 1000
                + indValues[0][0].length * 100 + ind0Values[0].length * 10
                + dimensionValues[0].length + generatedArrayValues[0].length
                + generatedIndexValues[0][0].length
                + generatedIndex1Values[0][0][0].length;
    }
}
`)

	outputs := convertJavaProjectDir(t, sourceRoot)
	moduleRoot := t.TempDir()
	goMod := "module collision\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => " + repoRootDir(t) + "\n"
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write collision go.mod: %v", err)
	}
	for relative, generated := range outputs {
		relative = strings.TrimPrefix(filepath.ToSlash(relative), "collision/")
		path := filepath.Join(moduleRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create generated collision package: %v", err)
		}
		if err := os.WriteFile(path, []byte(generated), 0o644); err != nil {
			t.Fatalf("write generated collision source: %v", err)
		}
	}
	appTest := `package app
import "testing"
func TestImportedArrayElementAliases(t *testing.T) {
    if got := Run(); got != 23446 { t.Fatalf("Run() = %d, want 23446", got) }
}
`
	if err := os.WriteFile(filepath.Join(moduleRoot, "app", "application_test.go"), []byte(appTest), 0o644); err != nil {
		t.Fatalf("write collision runtime test: %v", err)
	}
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = moduleRoot
	if output, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("tidy import-alias collision module:\n%s", output)
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = moduleRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated import-alias collision program failed:\n%s", output)
	}
}
