package transpiler

import (
	"strings"
	"testing"
)

func TestUserMethodOverloadsResolveByArityAndStaticArgumentType(t *testing.T) {
	src := `
public class OverloadProgram {
    public static String numeric(long value) { return "long"; }
    public static String numeric(float value) { return "float"; }
    public static String numeric(double value) { return "double"; }

    public static String reference(Object value) { return "object"; }
    public static String reference(String value) { return "string"; }

    public static int total(int a, int b) { return a + b; }
    public static int total(int a, int b, int c) { return a + b + c; }

    public String member(int value) { return "member-int"; }
    public String member(String value) { return "member-string"; }

    public static double needsDouble(double value) { return value; }

    public static String run() {
        int i = 7;
        long l = 8L;
        float f = 2.5F;
        double d = 3.5;
        String s = "typed";
        Object o = s;
        OverloadProgram program = new OverloadProgram();

        return numeric(i) + "," + numeric(l) + "," + numeric(f) + "," + OverloadProgram.numeric(d)
            + "," + reference("literal") + "," + reference(s) + "," + reference(o)
            + "," + total(1, 2) + "," + total(1, 2, 3)
            + "," + program.member(1) + "," + program.member(s)
            + "," + needsDouble(i);
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := normalizeSpaces(out)
	if !strings.Contains(flat, "float64(i)") {
		t.Fatalf("expected an explicit Go conversion for Java int-to-double method invocation widening, got:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestOverloadDispatchBehavior(t *testing.T) {
    const want = "long,long,float,double,string,string,object,3,6,member-int,member-string,7.0"
    if got := Run(); got != want {
        t.Fatalf("Run() = %q, want %q", got, want)
    }
}
`)
}

func TestOverloadResolutionUsesLocalDeclaredTypeInsteadOfInitializerType(t *testing.T) {
	src := `
public class DeclaredTypeProgram {
    public static String choose(Object value) { return "object"; }
    public static String choose(String value) { return "string"; }

    public static String run() {
        Object value = "text";
        return choose(value);
    }
}
`

	out := normalizeSpaces(renderGoFileFromJava(t, src))
	if !strings.Contains(out, "return Choose0(value)") {
		t.Fatalf("expected overload selection to use Object, the local's declared Java type, got:\n%s", out)
	}
}
