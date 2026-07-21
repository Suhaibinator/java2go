package transpiler

import (
	"strings"
	"testing"
)

func TestStaticFieldInitializersInterleaveWithStaticBlocks(t *testing.T) {
	src := `
public class StaticOrderProgram {
    static String trace = "";
    static int a = mark("a", 1);
    static int b;

    static {
        trace = trace + "block1,";
        b = mark("b", 2);
    }

    static int c = mark("c", 3);

    static {
        trace = trace + "block2,";
    }

    static int mark(String name, int value) {
        trace = trace + name + ",";
        return value;
    }

    public static String run() {
        return trace + (a + b + c);
    }
}
`

	out := renderGoFileFromJava(t, src)
	if strings.Count(out, "func init()") != 1 {
		t.Fatalf("expected one consolidated package initializer, got:\n%s", out)
	}
	lastPosition := -1
	for _, marker := range []string{
		`markJava2goExecution(__java2goExecution, "a", 1)`,
		`"block1,"`,
		`markJava2goExecution(__java2goExecution, "b", 2)`,
		`markJava2goExecution(__java2goExecution, "c", 3)`,
		`"block2,"`,
	} {
		position := strings.Index(out, marker)
		if position <= lastPosition {
			t.Fatalf("expected %q after the preceding Java initializer in generated output, got:\n%s", marker, out)
		}
		lastPosition = position
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestStaticInitializationOrder(t *testing.T) {
    if got := Run(); got != "a,block1,b,c,block2,6" {
        t.Fatalf("Run() = %q, want source-ordered Java initialization", got)
    }
}
`)
}
