package transpiler

import (
	"strings"
	"testing"
)

func TestObjectArrayInitializerBoxesPrimitiveExpressionsToJavaWidths(t *testing.T) {
	src := `
public class ObjectArrayBoxingProgram {
    static String kind(Object value) {
        if (value instanceof Integer i) {
            return "int:" + i;
        }
        if (value instanceof Long l) {
            return "long:" + l;
        }
        if (value instanceof Float f) {
            return "float:" + f;
        }
        if (value instanceof Double d) {
            return "double:" + d;
        }
        return "other";
    }

    public static String run() {
        Object[] values = new Object[] { 21, 9L, 1.5F, 2.5 };
		Object direct = 22;
        return kind(values[0]) + "," + kind(values[1]) + ","
			+ kind(values[2]) + "," + kind(values[3]) + "," + kind(direct);
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	for _, boxed := range []string{"[]any{int32(21), int64(9), float32(1.5), 2.5}", ".(int32)", ".(int64)", ".(float32)", ".(float64)"} {
		if !strings.Contains(flat, boxed) {
			t.Fatalf("expected Object[] autoboxing output to contain %q, got:\n%s", boxed, out)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestObjectArrayBoxingRuntime(t *testing.T) {
	if got := Run(); got != "int:21,long:9,float:1.5,double:2.5,int:22" {
        t.Fatalf("Run() = %q", got)
    }
}
`)
}
