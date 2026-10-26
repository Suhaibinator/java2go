package transpiler

import (
	"strings"
	"testing"
)

func TestNullableStringReferencesPreserveJavaSemantics(t *testing.T) {
	src := `
class NullableStringBase {
    String fromInitializer = inspect("base-init");
    String fromConstructor;

    NullableStringBase() {
        fromConstructor = inspect("base-ctor");
    }

    String inspect(String phase) {
        return phase + "=base";
    }
}

class NullableStringChild extends NullableStringBase {
    String value = initialize();

    String initialize() {
        return "ready";
    }

    String raw() {
        return value;
    }

    String inspect(String phase) {
        return phase + "=" + String.valueOf(value) + "/" + String.valueOf(raw());
    }
}

class NullableStringState {
    String implicit;
    String empty = "";
	String conditional = true ? null : "unused";

    int implicitLength() {
        return implicit.length();
    }
}

public class NullableStringReferenceProgram {
    static String staticValue;
	static String staticFirst = readStaticLater();
	static boolean staticFirstWasNull = staticFirst == null;
	static String staticLater = "later";
	static String staticObserved = String.valueOf(readStaticAfter());
	static String staticAfter = "after";

	static String readStaticLater() {
		return staticLater;
	}

	static String readStaticAfter() {
		return staticAfter;
	}

    static String identity(String value) {
        return value;
    }

    static String explicitNull() {
        return null;
    }

    public static String run() {
        NullableStringChild child = new NullableStringChild();
        NullableStringState state = new NullableStringState();

		String local = null;
		local += "x";
		state.implicit += "y";

		String first = null;
		String second = first;
		String third = (second);
		String castNull = (String) null;
		String selected = true ? third : "unused";
		String assigned;
		assigned = selected;

		boolean comparisons = first == null
		        && null == second
		        && !(third != null)
		        && !(null != castNull)
		        && first == (null)
		        && (null) == second;
		boolean castNullOnLeft = ((String) null) == null;
		boolean castNullOnRight = null == (String) null;
		boolean bothNullEqual = null == null;
		boolean bothNullNotEqual = null != null;

		String[] direct = new String[2];
		String[][] matrix = new String[2][2];
		String[][] partial = new String[1][];
		boolean arrayDefaults = direct[0] == null
		        && matrix[0][0] == null
		        && matrix[1][1] == null
		        && partial[0] == null;
		direct[0] += "array";

        String dereference = "missing-npe";
        try {
            new NullableStringState().implicitLength();
        } catch (NullPointerException expected) {
            dereference = "npe";
        }

        return child.fromInitializer + "|" + child.fromConstructor
		        + "|" + (staticValue == null)
		        + "/" + staticFirstWasNull
		        + "/" + String.valueOf(staticFirst)
		        + "/" + staticObserved
                + "/" + (state.implicit == null)
                + "/" + (state.empty == null)
		        + "/" + (state.conditional == null)
                + "|" + local + "/" + state.implicit
		        + "|" + comparisons
		        + "/" + castNullOnLeft
		        + "/" + castNullOnRight
		        + "/" + bothNullEqual
		        + "/" + bothNullNotEqual
		        + "/" + String.valueOf(assigned)
		        + "|" + arrayDefaults + "/" + direct[0]
                + "|" + String.valueOf(identity(null))
                + "/" + String.valueOf(explicitNull())
                + "|" + dereference;
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		`value *stdjava.JavaString`,
		`stdjava.JavaReferenceEqual(first, nil)`,
		`stdjava.JavaStringTextOperandExecution(__java2goExecution, old)`,
		`stdjava.RequireJavaString(ne.implicit).Length()`,
	} {
		if !strings.Contains(flat, want) {
			t.Fatalf("nullable String lowering missing %q:\n%s", want, out)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestNullableStringRuntime(t *testing.T) {
	const want = "base-init=null/null|base-ctor=null/null|true/true/null/null/false/false/true|nullx/nully|true/true/true/true/false/null|true/nullarray|null/null|npe"
	got := Run()
	if got == nil {
		t.Fatal("Run returned null")
	}
	if !slices.Equal(got.UTF16Copy(), utf16.Encode([]rune(want))) {
		t.Fatalf("Run() = UTF16 %x, want UTF16 %x", got.UTF16Copy(), utf16.Encode([]rune(want)))
	}
}
`)
}

func TestNullableStringRuntimeImportAliasAvoidsJavaIdentifiers(t *testing.T) {
	src := `
public class NullableStringAliasCollision {
    public static String run(String stdjava) {
        String stdjavapkg = null;
        String stdjava2 = stdjavapkg;
        stdjava2 += stdjava;
        return stdjava2;
    }
}
`

	out := renderGoFileFromJava(t, src)
	flat := strings.Join(strings.Fields(out), " ")
	if !strings.Contains(flat, `stdjava3 "github.com/NickyBoy89/java2go/stdjava"`) {
		t.Fatalf("runtime import did not avoid stdjava/stdjavapkg/stdjava2 declarations:\n%s", out)
	}
	if !strings.Contains(flat, "stdjava3.ConcatJavaStrings(stdjava3.JavaStringTextOperandExecution(__java2goExecution, old), stdjava3.JavaStringTextOperandExecution(__java2goExecution, rhs))") {
		t.Fatalf("nullable String calls did not use allocated runtime alias:\n%s", out)
	}

	runGoTestInTempModule(t, out, `
package main

import (
	"slices"
	"testing"
	"unicode/utf16"

	stdjava "github.com/NickyBoy89/java2go/stdjava"
)

func TestAliasCollisionRuntime(t *testing.T) {
	got := Run(stdjava.JavaStringFromHostUTF8("x"))
	if got == nil {
		t.Fatal("Run returned null")
	}
	if !slices.Equal(got.UTF16Copy(), utf16.Encode([]rune("nullx"))) {
		t.Fatalf("Run(x) = UTF16 %x, want nullx", got.UTF16Copy())
	}
}
`)
}
