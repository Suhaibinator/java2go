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
		{"length", "s.length()", "stdjava.RequireJavaString(s).Length()"},
		{"isEmpty", "s.isEmpty()", "stdjava.RequireJavaString(s).Length() == 0"},
		{"isBlank", "s.isBlank()", "stdjava.JavaStringIsBlank(stdjava.RequireJavaString(s))"},
		{"charAt", "s.charAt(2)", `stdjava.BoxCharacter(int32(func() rune {
			__java2goInvocationReceiver := s
			var __java2goInvocationArg0 int32 = 2
			return stdjava.RequireJavaString(__java2goInvocationReceiver).CharAt(__java2goInvocationArg0)
		}()))`},
		{"substring1", "s.substring(1)", `func() *stdjava.JavaString {
			__java2goInvocationReceiver := s
			var __java2goInvocationArg0 int32 = 1
			return stdjava.JavaStringSubstringFrom(stdjava.RequireJavaString(__java2goInvocationReceiver), __java2goInvocationArg0)
		}()`},
		{"substring2", "s.substring(1, 3)", `func() *stdjava.JavaString {
			__java2goInvocationReceiver := s
			var __java2goInvocationArg0 int32 = 1
			var __java2goInvocationArg1 int32 = 3
			return stdjava.RequireJavaString(__java2goInvocationReceiver).Substring(__java2goInvocationArg0, __java2goInvocationArg1)
		}()`},
		{"indexOf", "s.indexOf(\"x\")", `stdjava.BoxInteger(int32(func() int32 {
			__java2goInvocationReceiver := s
			__java2goInvocationArg0 := stdjava.JavaStringLiteralUTF16([]uint16{120})
			return stdjava.JavaStringIndexOf(stdjava.RequireJavaString(__java2goInvocationReceiver), __java2goInvocationArg0)
		}()))`},
		{"lastIndexOf", "s.lastIndexOf(\"x\")", `stdjava.BoxInteger(int32(func() int32 {
			__java2goInvocationReceiver := s
			__java2goInvocationArg0 := stdjava.JavaStringLiteralUTF16([]uint16{120})
			return stdjava.JavaStringLastIndexOf(stdjava.RequireJavaString(__java2goInvocationReceiver), __java2goInvocationArg0)
		}()))`},
		{"contains", "s.contains(\"x\")", `stdjava.BoxBoolean(func() bool {
			__java2goInvocationReceiver := s
			__java2goInvocationArg0 := stdjava.JavaStringLiteralUTF16([]uint16{120})
			return stdjava.JavaStringContainsExecution(__java2goExecution, stdjava.RequireJavaString(__java2goInvocationReceiver), __java2goInvocationArg0)
		}())`},
		{"startsWith", "s.startsWith(\"x\")", `stdjava.BoxBoolean(func() bool {
			__java2goInvocationReceiver := s
			__java2goInvocationArg0 := stdjava.JavaStringLiteralUTF16([]uint16{120})
			return stdjava.JavaStringStartsWith(stdjava.RequireJavaString(__java2goInvocationReceiver), __java2goInvocationArg0)
		}())`},
		{"endsWith", "s.endsWith(\"x\")", `stdjava.BoxBoolean(func() bool {
			__java2goInvocationReceiver := s
			__java2goInvocationArg0 := stdjava.JavaStringLiteralUTF16([]uint16{120})
			return stdjava.JavaStringEndsWith(stdjava.RequireJavaString(__java2goInvocationReceiver), __java2goInvocationArg0)
		}())`},
		{"equals", "s.equals(\"x\")", `stdjava.BoxBoolean(func() bool {
			__java2goInvocationReceiver := s
			__java2goInvocationArg0 := stdjava.JavaStringLiteralUTF16([]uint16{120})
			return stdjava.RequireJavaString(__java2goInvocationReceiver).Equals(__java2goInvocationArg0)
		}())`},
		{"equalsIgnoreCase", "s.equalsIgnoreCase(\"x\")", `stdjava.BoxBoolean(func() bool {
			__java2goInvocationReceiver := s
			__java2goInvocationArg0 := stdjava.JavaStringLiteralUTF16([]uint16{120})
			return stdjava.JavaStringEqualsIgnoreCase(stdjava.RequireJavaString(__java2goInvocationReceiver), __java2goInvocationArg0)
		}())`},
		{"compareTo", "s.compareTo(\"x\")", `stdjava.BoxInteger(int32(func() int32 {
			__java2goInvocationReceiver := s
			__java2goInvocationArg0 := stdjava.JavaStringLiteralUTF16([]uint16{120})
			return stdjava.RequireJavaString(__java2goInvocationReceiver).CompareTo(__java2goInvocationArg0)
		}()))`},
		{"toUpperCase", "s.toUpperCase()", "stdjava.JavaStringToUpperCase(stdjava.RequireJavaString(s))"},
		{"toLowerCase", "s.toLowerCase()", "stdjava.JavaStringToLowerCase(stdjava.RequireJavaString(s))"},
		{"trim", "s.trim()", "stdjava.JavaStringTrim(stdjava.RequireJavaString(s))"},
		{"strip", "s.strip()", "stdjava.JavaStringStrip(stdjava.RequireJavaString(s))"},
		{"replace", "s.replace(\"a\", \"b\")", `func() *stdjava.JavaString {
			__java2goInvocationReceiver := s
			__java2goInvocationArg0 := stdjava.JavaStringLiteralUTF16([]uint16{97})
			__java2goInvocationArg1 := stdjava.JavaStringLiteralUTF16([]uint16{98})
			return stdjava.JavaStringReplaceExecution(__java2goExecution, stdjava.RequireJavaString(__java2goInvocationReceiver), __java2goInvocationArg0, __java2goInvocationArg1)
		}()`},
		{"split", "s.split(\",\")", `func() *stdjava.ReferenceArray {
			__java2goInvocationReceiver := s
			__java2goInvocationArg0 := stdjava.JavaStringLiteralUTF16([]uint16{44})
			return stdjava.JavaStringSplitArray(stdjava.RequireJavaString(__java2goInvocationReceiver), __java2goInvocationArg0)
		}()`},
		{"chars", "s.chars()", "stdjava.JavaStringCharsStream(stdjava.RequireJavaString(s))"},
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
		{"valueOf", "String.valueOf(5)", "stdjava.JavaStringValueOfInt(int32(5))"},
		{"format", "String.format(\"%d\", 5)", "stdjava.JavaStringFormatExecution(__java2goExecution, stdjava.JavaStringLiteralUTF16([]uint16{37, 100}), stdjava.ReferenceArrayLiteralOf[any](stdjava.ObjectTypeID, stdjava.BoxInteger(int32(5))))"},
		{"join", "String.join(\",\", parts)", "stdjava.JavaStringJoinArrayExecution(__java2goExecution, stdjava.JavaStringLiteralUTF16([]uint16{44}), parts)"},
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
	assertContains(t, out, "sb.Append(stdjava.JavaStringTextOperandExecution(__java2goExecution, stdjava.JavaStringLiteralUTF16([]uint16{97})))")
	assertContains(t, out, "sb.Append(stdjava.JavaStringValueOfInt(int32(1)))")
	assertContains(t, out, "sb.Insert(0, stdjava.JavaStringTextOperandExecution(__java2goExecution, stdjava.JavaStringLiteralUTF16([]uint16{122})))")
	assertContains(t, out, "sb.Reverse()")
	assertContains(t, out, "sb.ToJavaString()")
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
		{"parseInt", "Integer.parseInt(s)", "stdjava.JavaIntegerParseInt(s)"},
		{"intToString", "Integer.toString(5)", "stdjava.JavaStringValueOfInt(5)"},
		// Generated helper ABI assertion only; independent JVM inputs/streams are unchanged.
		{"parseLong", "Long.parseLong(s)", "stdjava.JavaLongParseLong(s)"},
		{"parseDouble", "Double.parseDouble(s)", "stdjava.JavaDoubleParseDouble(s)"},
		{"parseBoolean", "Boolean.parseBoolean(s)", "stdjava.JavaBooleanParseBoolean(s)"},
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
	assertContains(t, out, `stdjava.JavaStringTrim(stdjava.JavaStringLiteralUTF16([]uint16{32, 32, 104, 105, 32, 32}))`)
	// split returns a descriptor-bearing String[]; array.length must use the
	// wrapper helper rather than native len so null/descriptor behavior survives.
	assertContains(t, out, `stdjava.ReferenceArrayLength(stdjava.JavaStringSplitArray(stdjava.JavaStringLiteralUTF16([]uint16{97, 44, 98, 44, 99}), stdjava.JavaStringLiteralUTF16([]uint16{44})))`)
	assertContains(t, out, `stdjava.ReferenceArrayGet[*stdjava.JavaString](stdjava.JavaStringSplitArray(stdjava.JavaStringLiteralUTF16([]uint16{97, 44, 98}), stdjava.JavaStringLiteralUTF16([]uint16{44})), 1, stdjava.StringTypeID)`)
	runGeneratedWithStdjava(t, out, `
package main

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestStringSplitArrayRuntime(t *testing.T) {
	trimmed := Trimmed()
	if trimmed == nil { t.Fatal("Trimmed() returned null") }
	if got := trimmed.UTF16Copy(); !slices.Equal(got, utf16.Encode([]rune("hi"))) {
		t.Fatalf("Trimmed() = %q, want hi", string(utf16.Decode(got)))
	}
	if got := Parts(); got != 3 {
		t.Fatalf("Parts() = %d, want 3", got)
	}
	if got := TrailingEmptyParts(); got != 2 {
		t.Fatalf("TrailingEmptyParts() = %d, want Java trailing-empty length 2", got)
	}
	second := Second()
	if second == nil { t.Fatal("Second() returned null") }
	if got := second.UTF16Copy(); !slices.Equal(got, utf16.Encode([]rune("b"))) {
		t.Fatalf("Second() = %q, want b", string(utf16.Decode(got)))
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
	assertContains(t, out, "stdjava.JavaStringToUpperCase(stdjava.JavaStringTrim(stdjava.RequireJavaString(s)))")
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

func TestBoxedTypesMapToObjects(t *testing.T) {
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
	assertContains(t, out, "NewBoxesJava2goExecution[*stdjava.Integer](__java2goExecution, stdjava.BoxInteger(int32(42)))")
	assertContains(t, out, "stdjava.BoxLong(")
	assertContains(t, out, "stdjava.BoxDouble(")
	assertContains(t, out, "stdjava.BoxBoolean(")
	if strings.Contains(out, "*Integer") || strings.Contains(out, "*Long") || strings.Contains(out, "*Double") || strings.Contains(out, "*Boolean") {
		t.Fatalf("boxed type leaked as an undefined pointer type:\n%s", out)
	}
}
