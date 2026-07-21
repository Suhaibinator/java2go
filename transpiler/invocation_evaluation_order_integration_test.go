package transpiler

import (
	"strings"
	"testing"
)

func TestGeneratedCode_RuntimeBehavior_InvocationEvaluationOrder(t *testing.T) {
	src := `
package invocationorder;

public class InvocationOrderProgram {
    static String trace = "";

    static final class Target {
        int instanceCall(int value) {
            trace = trace + "m";
            return value;
        }

        static int staticCall(int value) {
            trace = trace + "s";
            return value;
        }
    }

    static Target receiver() {
        trace = trace + "r";
        return null;
    }

    static int argument() {
        trace = trace + "a";
        return 7;
    }

    public static String runNullInstanceCall() {
        trace = "";
        try {
            receiver().instanceCall(argument());
        } catch (NullPointerException expected) {
            trace = trace + "c";
        }
        return trace;
    }

    @SuppressWarnings("static-access")
    public static String runStaticCallThroughExpression() {
        trace = "";
        int result = receiver().staticCall(argument());
        return result + ":" + trace;
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := strings.Join(strings.Fields(out), " ")
	if !strings.Contains(flat, "receiver().instanceCall(argument())") {
		t.Fatalf("ordinary instance invocation should retain Go's natural receiver-then-arguments order:\n%s", out)
	}
	if !strings.Contains(flat, "if it == nil { _ = *it }") {
		t.Fatalf("source-backed instance method must reject nil before its Java body:\n%s", out)
	}
	if !strings.Contains(flat, "_ = receiver() return staticCall(argument())") {
		t.Fatalf("static invocation must evaluate its qualifying expression before arguments:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestInvocationEvaluationOrder(t *testing.T) {
    if got := RunNullInstanceCall(); got != "rac" {
        t.Fatalf("null instance call trace = %q, want receiver, argument, catch", got)
    }
    if got := RunStaticCallThroughExpression(); got != "7:ras" {
        t.Fatalf("static-through-expression trace = %q, want receiver, argument, body", got)
    }
}
`)
}
