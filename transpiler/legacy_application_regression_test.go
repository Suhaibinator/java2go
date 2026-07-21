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
	if !strings.Contains(out, "stdjava.ArrayDeepToString(values)") {
		t.Fatalf("expected Arrays.deepToString to use the stdjava runtime helper, got:\n%s", out)
	}
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestDeepArrayRendering(t *testing.T) {
    const want = "[[0, 0, 0], [0, 0, 0]]"
    if got := Run(); got != want {
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

import "testing"

func TestKeywordMethodNames(t *testing.T) {
    const want = "field:range:value"
    if got := Run(); got != want {
        t.Fatalf("Run() = %q, want %q", got, want)
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
	if !strings.Contains(out, "stdjava.NewMap[string, int32]()") {
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
