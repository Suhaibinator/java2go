package stdjava

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Frozen before production helpers; the external gate compares this complete
// transcript to the explicit JDK21 oracle before accepting any Go behavior.
func TestCampaignJavaStringCharsJDK21(t *testing.T) {
	var output strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&output, format+"\n", args...) }
	units := func(label string, chars *PrimitiveArray[rune]) {
		fmt.Fprintf(&output, "%s=", label)
		if chars == nil || chars.ComponentType() != PrimitiveTypeID("char") {
			t.Fatal("invalid char array descriptor")
		}
		for _, c := range chars.Elements {
			fmt.Fprintf(&output, "%04x,", c)
		}
		output.WriteByte('\n')
	}
	exception := func(label string, action func(), message bool) {
		defer func() {
			err := recover()
			if err == nil {
				line("%s=NO_EXCEPTION", label)
				return
			}
			typed, ok := err.(interface {
				ThrowableTypeName() string
				Message() string
			})
			if !ok {
				t.Fatalf("%s: non-Java panic %T: %v", label, err, err)
			}
			if message {
				line("%s=%s:%s", label, typed.ThrowableTypeName(), typed.Message())
			} else {
				line("%s=%s", label, typed.ThrowableTypeName())
			}
		}()
		action()
	}
	chars := PrimitiveArrayLiteral[rune](PrimitiveTypeID("char"), 'A', 0xD800, 0, 0xDC00, 0xD83D, 0xDE00, 0xFFFF)
	whole, copy := JavaStringFromChars(chars), JavaStringFromChars(chars)
	part := JavaStringFromCharsRange(chars, 1, 5)
	line("construct.identity=%t", whole != copy && whole.Equals(copy))
	chars.Elements[1] = 'X'
	units("construct.copy", JavaStringToCharArray(whole))
	units("range.copy", JavaStringToCharArray(part))
	first, second := JavaStringToCharArray(whole), JavaStringToCharArray(whole)
	first.Elements[0] = 'Z'
	line("array.identity=%t", first != second)
	units("array.copy", second)
	units("string.unchanged", JavaStringToCharArray(whole))
	empty1 := JavaStringFromChars(NewPrimitiveArray[rune](0, PrimitiveTypeID("char")))
	empty2 := JavaStringFromCharsRange(chars, 7, 0)
	literal := JavaStringLiteralUTF16(nil)
	line("empty.identity=%t", empty1 != empty2 && empty1 != literal && empty2 != literal)
	emptyCharsFirst := JavaStringToCharArray(empty1)
	emptyCharsSecond := JavaStringToCharArray(empty1)
	if emptyCharsFirst == nil || emptyCharsSecond == nil || len(emptyCharsFirst.Elements) != 0 || len(emptyCharsSecond.Elements) != 0 || emptyCharsFirst.ComponentType() != PrimitiveTypeID("char") || emptyCharsSecond.ComponentType() != PrimitiveTypeID("char") {
		t.Fatal("empty toCharArray must return nonnull zero-length char arrays")
	}
	line("empty.array.identity=%t", emptyCharsFirst != emptyCharsSecond)
	line("substring.full.identity=%t", JavaStringSubstringFrom(whole, 0) == whole)
	line("substring.empty.identity=%t", JavaStringSubstringFrom(whole, whole.Length()) == literal)
	line("substring.empty.full.identity=%t", JavaStringSubstringFrom(empty1, 0) == empty1)
	units("substring.units", JavaStringToCharArray(JavaStringSubstringFrom(whole, 1)))
	exception("null.whole", func() { JavaStringFromChars(nil) }, true)
	exception("null.range", func() { JavaStringFromCharsRange(nil, -1, -1) }, true)
	exception("null.array", func() { JavaStringToCharArray(nil) }, false)
	exception("null.substring", func() { JavaStringSubstringFrom(nil, -1) }, false)
	for _, r := range [][2]int32{{-1, 0}, {0, -1}, {8, 0}, {7, 1}, {1, 7}, {2147483647, 1}, {1, 2147483647}, {2147483647, 2147483647}, {-2147483648, -2147483648}} {
		exception(fmt.Sprintf("range.%d.%d", r[0], r[1]), func() { JavaStringFromCharsRange(chars, r[0], r[1]) }, true)
	}
	for _, begin := range []int32{-1, 8, 2147483647, -2147483648} {
		exception(fmt.Sprintf("substring.%d", begin), func() { JavaStringSubstringFrom(whole, begin) }, true)
	}
	oraclePath := os.Getenv("JAVA2GO_CHARS_ORACLE")
	if oraclePath == "" {
		oraclePath = "testdata/string_chars_jdk21.txt"
	}
	oracle, err := os.ReadFile(oraclePath)
	if err != nil {
		t.Fatal(err)
	}
	if output.String() != string(oracle) {
		t.Fatalf("JDK21 transcript mismatch\nGo:\n%s\nJDK21:\n%s", output.String(), oracle)
	}
}
