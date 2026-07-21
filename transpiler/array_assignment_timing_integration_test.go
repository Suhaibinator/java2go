package transpiler

import (
	"strings"
	"testing"
)

func TestSimpleArrayAssignmentEvaluatesRHSBeforeStoreChecks(t *testing.T) {
	src := `
public class ArrayAssignmentTimingProgram {
    private static String trace = "";
    private static final int[] valid = new int[3];

    private static int[] array(int mode) {
        trace = trace + "a";
        if (mode == 0) {
            return null;
        }
        if (mode == 1) {
            return new int[1];
        }
        if (mode == 2) {
            return new int[0];
        }
        return valid;
    }

    private static int index(int mode) {
        trace = trace + "i";
        if (mode == 1) {
            throw new IllegalStateException("index");
        }
        return 2;
    }

    private static int rhs(int mode) {
        trace = trace + "r";
        if (mode == 1) {
            throw new IllegalArgumentException("rhs");
        }
        return 7;
    }

    public static String nullOrder() {
        trace = "";
        try {
            array(0)[index(0)] = rhs(0);
        } catch (NullPointerException expected) {
            trace = trace + "N";
        } catch (ArrayIndexOutOfBoundsException wrong) {
            trace = trace + "B";
        }
        return trace;
    }

    public static String boundsOrder() {
        trace = "";
        try {
            array(1)[index(0)] = rhs(0);
        } catch (NullPointerException wrong) {
            trace = trace + "N";
        } catch (ArrayIndexOutOfBoundsException expected) {
            trace = trace + "B";
        }
        return trace;
    }

    public static String emptyBoundsOrder() {
        trace = "";
        try {
            array(2)[index(0)] = rhs(0);
        } catch (NullPointerException wrong) {
            trace = trace + "N";
        } catch (ArrayIndexOutOfBoundsException expected) {
            trace = trace + "B";
        }
        return trace;
    }

    public static String rhsExceptionWinsOverNullCheck() {
        trace = "";
        try {
            array(0)[index(0)] = rhs(1);
        } catch (IllegalArgumentException expected) {
            trace = trace + "R";
        } catch (NullPointerException wrong) {
            trace = trace + "N";
        }
        return trace;
    }

    public static String indexExceptionSkipsRhs() {
        trace = "";
        try {
            array(0)[index(1)] = rhs(0);
        } catch (IllegalStateException expected) {
            trace = trace + "I";
        }
        return trace;
    }

    public static String expressionValue() {
        trace = "";
        int assigned = (array(3)[index(0)] = rhs(0));
        return trace + ":" + assigned + ":" + valid[2];
    }

    public static String expressionNullOrder() {
        trace = "";
        try {
            int ignored = (array(0)[index(0)] = rhs(0));
        } catch (NullPointerException expected) {
            trace = trace + "N";
        } catch (ArrayIndexOutOfBoundsException wrong) {
            trace = trace + "B";
        }
        return trace;
    }

    public static String expressionBoundsOrder() {
        trace = "";
        try {
            int ignored = (array(1)[index(0)] = rhs(0));
        } catch (NullPointerException wrong) {
            trace = trace + "N";
        } catch (ArrayIndexOutOfBoundsException expected) {
            trace = trace + "B";
        }
        return trace;
    }

    public static String compoundStillUsesItsExistingPath() {
        trace = "";
        valid[2] = 10;
        array(3)[index(0)] += rhs(0);
        return trace + ":" + valid[2];
    }
}
`

	out := renderGoFileFromJava(t, src)
	for _, functionName := range []string{"NullOrder", "ExpressionNullOrder"} {
		function := generatedFunctionText(out, functionName)
		if !strings.Contains(function, "stdjava.ArraySet(arrayJava2goExecution(__java2goExecution,") {
			t.Fatalf("%s must stage its simple array store through ArraySet:\n%s", functionName, function)
		}
	}
	compound := generatedFunctionText(out, "CompoundStillUsesItsExistingPath")
	if strings.Contains(compound, "ArraySet(arrayJava2goExecution(__java2goExecution, 3)") ||
		!strings.Contains(compound, "func(dst *int32)") {
		t.Fatalf("compound array assignment must retain its existing lowering:\n%s", compound)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestArrayAssignmentTimingRuntime(t *testing.T) {
    cases := []struct {
        name string
        call func() string
        want string
    }{
        {"null", NullOrder, "airN"},
        {"bounds", BoundsOrder, "airB"},
        {"empty-bounds", EmptyBoundsOrder, "airB"},
        {"rhs-before-null", RhsExceptionWinsOverNullCheck, "airR"},
        {"index-before-rhs", IndexExceptionSkipsRhs, "aiI"},
        {"expression-value", ExpressionValue, "air:7:7"},
        {"expression-null", ExpressionNullOrder, "airN"},
        {"expression-bounds", ExpressionBoundsOrder, "airB"},
        {"compound-path", CompoundStillUsesItsExistingPath, "air:17"},
    }
    for _, tc := range cases {
        if got := tc.call(); got != tc.want {
            t.Errorf("%s = %q, want %q", tc.name, got, tc.want)
        }
    }
}
`)
}
