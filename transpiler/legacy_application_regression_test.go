package transpiler

import (
	"strings"
	"testing"
)

func TestArraysDeepToStringIntrinsicRuntime(t *testing.T) {
	src := `
import java.util.Arrays;
public class DeepArrayProgram {
    public static String run() {
        int[][] values = new int[2][3];
        return Arrays.deepToString(values);
    }
}
`

	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "stdjava.JavaArrayDeepToStringExecution(__java2goExecution, values)") {
		t.Fatalf("expected Arrays.deepToString to use the stdjava runtime helper, got:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main

import ("testing"; "unicode/utf16")

func TestDeepArrayRendering(t *testing.T) {
    const want = "[[0, 0, 0], [0, 0, 0]]"
    reference := Run()
    if reference == nil { t.Fatal("Run() returned null") }
    if got := string(utf16.Decode(reference.UTF16Copy())); got != want {
        t.Fatalf("Run() = %q, want %q", got, want)
    }
}
`)
}

func TestKeywordNamedMethodsResolveOriginalSymbols(t *testing.T) {
	src := `
public class KeywordMethodProgram {
    private String map = "initial";
    private String range() { return "range"; }
    public String type(String type) { return type; }

    public static String run() {
        KeywordMethodProgram program = new KeywordMethodProgram();
        program.map = "field";
        return program.map + ":" + program.range() + ":" + program.type("value");
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestKeywordMethodNames(t *testing.T) {
    const want = "field:range:value"
    reference := Run()
    if reference == nil { t.Fatal("Run() returned null") }
    if got := reference.UTF16Copy(); !slices.Equal(got, utf16.Encode([]rune(want))) {
        t.Fatalf("Run() = %q, want %q", string(utf16.Decode(got)), want)
    }
}
`)
}

func TestDiamondAssignmentUsesLeftHandSideGenericType(t *testing.T) {
	src := `
import java.util.HashMap;
import java.util.Map;
public class DiamondAssignmentProgram {
    private Map<String, Integer> values;

    public DiamondAssignmentProgram() {
        this.values = new HashMap<>();
    }

    public static int run() {
        DiamondAssignmentProgram program = new DiamondAssignmentProgram();
        program.values.put("answer", 42);
        return program.values.get("answer");
    }
}
`

	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "stdjava.NewMap[*stdjava.JavaString, *stdjava.Integer]()") {
		t.Fatalf("expected assignment-target generics to type the diamond constructor, got:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestDiamondAssignment(t *testing.T) {
    if got := Run(); got != 42 {
        t.Fatalf("Run() = %d, want 42", got)
    }
}
`)
}

func TestCompoundAssignmentInfersArrayLengthType(t *testing.T) {
	src := `
public class ArrayLengthCompoundProgram {
    public static int run() {
        int total = 1;
        int[] values = new int[4];
        total += values.length;
        return total;
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "BadExpr") {
		t.Fatalf("array length must retain its Java int type in compound assignment, got:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestArrayLengthCompoundAssignment(t *testing.T) {
    if got := Run(); got != 5 {
        t.Fatalf("Run() = %d, want 5", got)
    }
}
`)
}
