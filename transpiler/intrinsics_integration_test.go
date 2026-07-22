package transpiler

import (
	"strings"
	"testing"
)

// renderMethodBody transpiles a single-method class and returns the normalized
// Go output, so intrinsic rewrites can be asserted against the generated source.
func renderIntrinsicProgram(t *testing.T, src string) string {
	t.Helper()
	out := renderGoFileFromJava(t, src)
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected transpiler to produce non-empty Go output")
	}
	return out
}

func assertContains(t *testing.T, out, want string) {
	t.Helper()
	if !strings.Contains(normalizeSpaces(out), normalizeSpaces(want)) {
		t.Fatalf("expected output to contain %q, got:\n%s", want, out)
	}
}

func TestIntrinsics_StringMethods(t *testing.T) {
	cases := []struct {
		name string
		expr string
		want string
	}{
		{"length", "s.length()", "stdjava.StringLength(stdjava.StringRequireNonNull(s))"},
		{"isEmpty", "s.isEmpty()", "len(stdjava.StringRequireNonNull(s)) == 0"},
		{"isBlank", "s.isBlank()", "stdjava.StringIsBlank(stdjava.StringRequireNonNull(s))"},
		{"charAt", "s.charAt(2)", "stdjava.StringCharAt(stdjava.StringRequireNonNull(s), 2)"},
		{"substring1", "s.substring(1)", "stdjava.StringSubstring(stdjava.StringRequireNonNull(s), 1)"},
		{"substring2", "s.substring(1, 3)", "stdjava.StringSubstringRange(stdjava.StringRequireNonNull(s), 1, 3)"},
		{"indexOf", "s.indexOf(\"x\")", "stdjava.StringIndexOf(stdjava.StringRequireNonNull(s), \"x\")"},
		{"lastIndexOf", "s.lastIndexOf(\"x\")", "stdjava.StringLastIndexOf(stdjava.StringRequireNonNull(s), \"x\")"},
		{"contains", "s.contains(\"x\")", "strings.Contains(stdjava.StringRequireNonNull(s), \"x\")"},
		{"startsWith", "s.startsWith(\"x\")", "strings.HasPrefix(stdjava.StringRequireNonNull(s), \"x\")"},
		{"endsWith", "s.endsWith(\"x\")", "strings.HasSuffix(stdjava.StringRequireNonNull(s), \"x\")"},
		{"equals", "s.equals(\"x\")", "stdjava.StringRequireNonNull(s) == \"x\""},
		{"equalsIgnoreCase", "s.equalsIgnoreCase(\"x\")", "stdjava.StringEqualsIgnoreCase(stdjava.StringRequireNonNull(s), \"x\")"},
		{"compareTo", "s.compareTo(\"x\")", "stdjava.StringCompareTo(stdjava.StringRequireNonNull(s), \"x\")"},
		{"toUpperCase", "s.toUpperCase()", "strings.ToUpper(stdjava.StringRequireNonNull(s))"},
		{"toLowerCase", "s.toLowerCase()", "strings.ToLower(stdjava.StringRequireNonNull(s))"},
		{"trim", "s.trim()", "strings.TrimSpace(stdjava.StringRequireNonNull(s))"},
		{"strip", "s.strip()", "strings.TrimSpace(stdjava.StringRequireNonNull(s))"},
		{"replace", "s.replace(\"a\", \"b\")", "stdjava.StringReplace(stdjava.StringRequireNonNull(s), \"a\", \"b\")"},
		{"split", "s.split(\",\")", "stdjava.StringSplitArray(stdjava.StringRequireNonNull(s), \",\")"},
		{"chars", "s.chars()", "stdjava.StringChars(stdjava.StringRequireNonNull(s))"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := `
public class StringIntrinsics {
    public static Object run(String s) {
        return ` + tc.expr + `;
    }
}
`
			out := renderIntrinsicProgram(t, src)
			assertContains(t, out, tc.want)
		})
	}
}

func TestIntrinsics_StringStatics(t *testing.T) {
	cases := []struct {
		name string
		expr string
		want string
	}{
		{"valueOf", "String.valueOf(5)", "stdjava.StringValueOf(5)"},
		{"format", "String.format(\"%d\", 5)", "fmt.Sprintf(\"%d\", 5)"},
		{"join", "String.join(\",\", parts)", "strings.Join(parts, \",\")"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := `
public class StringStatics {
    public static Object run(String[] parts) {
        return ` + tc.expr + `;
    }
}
`
			out := renderIntrinsicProgram(t, src)
			assertContains(t, out, tc.want)
		})
	}
}

func TestIntrinsics_StringBuilder(t *testing.T) {
	src := `
public class SBProgram {
    public static String run() {
        StringBuilder sb = new StringBuilder();
        sb.append("a");
        sb.append(1);
        sb.insert(0, "z");
        sb.reverse();
        return sb.toString();
    }
}
`
	out := renderIntrinsicProgram(t, src)
	assertContains(t, out, "sb.Append(\"a\")")
	assertContains(t, out, "sb.Append(1)")
	assertContains(t, out, "sb.Insert(0, \"z\")")
	assertContains(t, out, "sb.Reverse()")
	assertContains(t, out, "sb.String()")
	assertContains(t, out, "stdjava.NewStringBuilder()")
}

func TestIntrinsics_Math(t *testing.T) {
	cases := []struct {
		name string
		expr string
		want string
	}{
		{"abs", "Math.abs(x)", "stdjava.MathAbs(x)"},
		{"max", "Math.max(x, 1)", "stdjava.MathMax(x, 1)"},
		{"min", "Math.min(x, 1)", "stdjava.MathMin(x, 1)"},
		{"pow", "Math.pow(2.0, 3.0)", "math.Pow(2.0, 3.0)"},
		{"sqrt", "Math.sqrt(9.0)", "math.Sqrt(9.0)"},
		{"floor", "Math.floor(1.5)", "math.Floor(1.5)"},
		{"ceil", "Math.ceil(1.5)", "math.Ceil(1.5)"},
		{"round", "Math.round(1.5)", "stdjava.MathRound(1.5)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := `
public class MathProgram {
    public static Object run(int x) {
        return ` + tc.expr + `;
    }
}
`
			out := renderIntrinsicProgram(t, src)
			assertContains(t, out, tc.want)
		})
	}
}

func TestIntrinsics_MathConstants(t *testing.T) {
	src := `
public class MathConstProgram {
    public static double run() {
        return Math.PI + Math.E;
    }
}
`
	out := renderIntrinsicProgram(t, src)
	assertContains(t, out, "math.Pi")
	assertContains(t, out, "math.E")
}

func TestIntrinsics_BoxedTypes(t *testing.T) {
	cases := []struct {
		name string
		expr string
		want string
	}{
		{"parseInt", "Integer.parseInt(s)", "stdjava.ParseInt(s)"},
		{"intToString", "Integer.toString(5)", "fmt.Sprint(5)"},
		{"parseLong", "Long.parseLong(s)", "stdjava.ParseLong(s)"},
		{"parseDouble", "Double.parseDouble(s)", "stdjava.ParseDouble(s)"},
		{"parseBoolean", "Boolean.parseBoolean(s)", "stdjava.ParseBoolean(s)"},
		{"isDigit", "Character.isDigit(c)", "stdjava.CharIsDigit(c)"},
		{"isLetter", "Character.isLetter(c)", "stdjava.CharIsLetter(c)"},
		{"charToUpper", "Character.toUpperCase(c)", "stdjava.CharToUpperCase(c)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := `
public class BoxedProgram {
    public static Object run(String s, char c) {
        return ` + tc.expr + `;
    }
}
`
			out := renderIntrinsicProgram(t, src)
			assertContains(t, out, tc.want)
		})
	}
}

func TestIntrinsics_BoxedConstants(t *testing.T) {
	src := `
public class BoxedConstProgram {
    public static int run() {
        return Integer.MAX_VALUE - Integer.MIN_VALUE;
    }
}
`
	out := renderIntrinsicProgram(t, src)
	assertContains(t, out, "math.MaxInt32")
	assertContains(t, out, "math.MinInt32")
}

func TestIntrinsics_UserMethodNotRewritten(t *testing.T) {
	// A user-defined method whose name collides with an intrinsic (length) must
	// still resolve to the user method, not the String intrinsic.
	src := `
public class Holder {
    public int length() {
        return 3;
    }
    public static int run() {
        Holder h = new Holder();
        return h.length();
    }
}
`
	out := renderIntrinsicProgram(t, src)
	if strings.Contains(out, "int32(len(") {
		t.Fatalf("user method length() was incorrectly rewritten as a String intrinsic:\n%s", out)
	}
	assertContains(t, out, "h.LengthJava2goExecution(__java2goExecution)")
}

func TestIntrinsics_StringLiteralReceiver(t *testing.T) {
	// Intrinsics fire when the receiver is a String literal, not just a variable.
	src := `
public class LiteralReceiver {
    public static String trimmed() {
        return "  hi  ".trim();
    }
    public static int parts() {
        return "a,b,c".split(",").length;
    }
	public static int trailingEmptyParts() {
		return "a,b,,".split(",").length;
	}
	public static String second() {
		return "a,b".split(",")[1];
	}
	public static int descriptorChecks() {
		Object parts = "a,b".split(",");
		int bits = 0;
		if (parts instanceof String[]) bits |= 1;
		if (parts instanceof Object[]) bits |= 2;
		return bits;
	}
}
`
	out := renderIntrinsicProgram(t, src)
	assertContains(t, out, `strings.TrimSpace("  hi  ")`)
	// split returns a descriptor-bearing String[]; array.length must use the
	// wrapper helper rather than native len so null/descriptor behavior survives.
	assertContains(t, out, `stdjava.ReferenceArrayLength(stdjava.StringSplitArray("a,b,c", ","))`)
	assertContains(t, out, `stdjava.ReferenceArrayGet[string](stdjava.StringSplitArray("a,b", ","), 1, stdjava.StringTypeID)`)
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestStringSplitArrayRuntime(t *testing.T) {
	if got := Trimmed(); got != "hi" {
		t.Fatalf("Trimmed() = %q, want hi", got)
	}
	if got := Parts(); got != 3 {
		t.Fatalf("Parts() = %d, want 3", got)
	}
	if got := TrailingEmptyParts(); got != 2 {
		t.Fatalf("TrailingEmptyParts() = %d, want Java trailing-empty length 2", got)
	}
	if got := Second(); got != "b" {
		t.Fatalf("Second() = %q, want b", got)
	}
	if got := DescriptorChecks(); got != 3 {
		t.Fatalf("DescriptorChecks() = %d, want String[] and covariant Object[] bits", got)
	}
}
`)
}

func TestIntrinsics_ChainedStringCalls(t *testing.T) {
	src := `
public class Chained {
    public static String run(String s) {
        return s.trim().toUpperCase();
    }
}
`
	out := renderIntrinsicProgram(t, src)
	assertContains(t, out, "strings.ToUpper(strings.TrimSpace(stdjava.StringRequireNonNull(s)))")
}

func TestJavaStdlibImportsStripped(t *testing.T) {
	// java.* / javax.* packages must never be emitted as Go imports (an
	// `import "java/util"` is an invalid path).
	src := `
import java.util.List;
import java.util.Map;
public class Importer {
    public List<String> items;
    public void run() {
        String s = "hi";
        s.length();
    }
}
`
	out := renderGoFileFromJava(t, src)
	if strings.Contains(out, "java/util") || strings.Contains(out, `"java"`) {
		t.Fatalf("java.* import leaked into generated output:\n%s", out)
	}
}

func TestBoxedTypesMapToPrimitives(t *testing.T) {
	src := `
public class Boxes<T> {
    private T value;
    public Boxes(T v) { this.value = v; }
    public static void run() {
        Boxes<Integer> b = new Boxes<Integer>(42);
        Long l = 5L;
        Double d = 1.5;
        Boolean flag = true;
    }
}
`
	out := renderGoFileFromJava(t, src)
	assertContains(t, out, "NewBoxesJava2goExecution[int32](__java2goExecution, int32(42))")
	if strings.Contains(out, "*Integer") || strings.Contains(out, "*Long") || strings.Contains(out, "*Double") || strings.Contains(out, "*Boolean") {
		t.Fatalf("boxed type leaked as an undefined pointer type:\n%s", out)
	}
}
