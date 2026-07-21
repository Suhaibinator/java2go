package transpiler

import (
	"strings"
	"testing"
)

func TestEnhancedForVarElementTypeDrivesCompoundAssignmentRuntime(t *testing.T) {
	src := `
public class EnhancedForVarProgram {
    public static int run() {
        var total = 0;
        int[] values = new int[] {10, 20, 30};
        for (var value : values) {
            total += value;
        }
        return total;
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "BadExpr") {
		t.Fatalf("enhanced-for var element type must be available to compound assignment, got:\n%s", out)
	}
	if !strings.Contains(normalizeSpaces(out), "func(rhs int32) int32") {
		t.Fatalf("expected the inferred int array element to type the compound assignment RHS, got:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestEnhancedForVarRuntime(t *testing.T) {
    if got := Run(); got != 60 {
        t.Fatalf("Run() = %d, want 60", got)
    }
}
`)
}
