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
	if !strings.Contains(out, `stdjava.NewClassInitialization("StaticOrderProgram")`) {
		t.Fatalf("expected lazy class-initialization state, got:\n%s", out)
	}
	if strings.Count(out, "func StaticOrderProgramJava2goEnsureInitialized(") != 1 {
		t.Fatalf("expected one lazy class-initialization entry point, got:\n%s", out)
	}
	if strings.Contains(out, "func init()") {
		t.Fatalf("static initialization should be lazy rather than run from a package initializer, got:\n%s", out)
	}
	lastPosition := -1
	for _, marker := range []string{
		`markJava2goExecution(__java2goExecution, stdjava.JavaStringLiteralUTF16([]uint16{97}), 1)`,
		`stdjava.JavaStringLiteralUTF16([]uint16{98, 108, 111, 99, 107, 49, 44})`,
		`markJava2goExecution(__java2goExecution, stdjava.JavaStringLiteralUTF16([]uint16{98}), 2)`,
		`markJava2goExecution(__java2goExecution, stdjava.JavaStringLiteralUTF16([]uint16{99}), 3)`,
		`stdjava.JavaStringLiteralUTF16([]uint16{98, 108, 111, 99, 107, 50, 44})`,
	} {
		position := strings.Index(out, marker)
		if position <= lastPosition {
			t.Fatalf("expected %q after the preceding Java initializer in generated output, got:\n%s", marker, out)
		}
		lastPosition = position
	}

	runGoTestInTempModule(t, out, `
package main

import (
    "slices"
    "testing"
    "unicode/utf16"
)

func TestStaticInitializationOrder(t *testing.T) {
    got := Run()
    if got == nil {
        t.Fatal("Run() returned null, want source-ordered Java initialization")
    }
    const want = "a,block1,b,c,block2,6"
    if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune(want))) {
        t.Fatalf("Run() UTF16 = %v, want %q for source-ordered Java initialization", units, want)
    }
}
`)
}
