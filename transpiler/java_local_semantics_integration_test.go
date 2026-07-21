package transpiler

import (
	"strings"
	"testing"
)

func TestJavaStringConversionRuntimeForCompoundAndNullConcatenation(t *testing.T) {
	src := `
public class JavaStringConversionProgram {
    public static String run() {
        String result = "value=";
        result += 7;
        result += ':';
        result += true;
        String nullable = null;
        return result + ":" + nullable;
    }
}
`

	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "stdjava.StringValueOf") {
		t.Fatalf("expected Java string conversion bridge in generated code, got:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestJavaStringConversion(t *testing.T) {
    const want = "value=7:true:null"
    if got := Run(); got != want {
        t.Fatalf("Run() = %q, want %q", got, want)
    }
}
`)
}

func TestJavaUnusedLocalsAndNullPrintRuntime(t *testing.T) {
	src := `
public class JavaLocalRulesProgram {
    public static void main(String[] args) {
        float unusedFloat = 1F;
        double unusedDouble = 1D;
        long unusedLong = 1L;
        int unusedInt = 1;

        int e;
        int f;
        for (e = 1, f = 1; e < 3; e++) {
        }

		String nullable = null;
		System.out.print(nullable);
		boolean wasNull = nullable == null;
		int resetLength = (nullable = "go").length();
		String assigned = (nullable += "!");
		System.out.print(":");
		System.out.print(wasNull);
		System.out.print(":");
		System.out.print(resetLength);
		System.out.print(":");
		System.out.print(nullable.length());
		System.out.print(":");
		System.out.print(assigned.concat("?"));
		System.out.print(":");
		System.out.println(e);
    }
}
`

	out := renderGoFileFromJava(t, src)
	for _, local := range []string{"unusedFloat", "unusedDouble", "unusedLong", "unusedInt", "f"} {
		if !strings.Contains(out, "_ = "+local) {
			t.Fatalf("expected generated discard for Java local %q, got:\n%s", local, out)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import (
    "io"
    "os"
    "testing"
)

func TestJavaLocalRules(t *testing.T) {
    original := os.Stdout
    reader, writer, err := os.Pipe()
    if err != nil {
        t.Fatal(err)
    }
    os.Stdout = writer
    Main()
    if err := writer.Close(); err != nil {
        t.Fatal(err)
    }
    os.Stdout = original

    output, err := io.ReadAll(reader)
    if err != nil {
        t.Fatal(err)
    }
	if got, want := string(output), "null:true:2:3:go!?:3\n"; got != want {
        t.Fatalf("Main output = %q, want %q", got, want)
    }
}
`)
}

func TestNullInitializedClassReferenceKeepsPointerAssignmentSemantics(t *testing.T) {
	src := `
public class NullableReferenceProgram {
    private int value;

    public NullableReferenceProgram(int value) {
        this.value = value;
    }

    public int get() {
        return this.value;
    }

    public static int run() {
        NullableReferenceProgram reference = null;
        if (reference != null) {
            return -1;
        }
        return (reference = new NullableReferenceProgram(7)).get();
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestNullableReferenceAssignment(t *testing.T) {
    if got := Run(); got != 7 {
        t.Fatalf("Run() = %d, want 7", got)
    }
}
`)
}
