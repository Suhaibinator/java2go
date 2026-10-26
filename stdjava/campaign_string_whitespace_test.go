package stdjava

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// Frozen before production implementation. The fixture is captured from the
// explicit JDK21 program, which exhaustively enumerates Character.isWhitespace.
func TestCampaignJavaStringWhitespaceJDK21(t *testing.T) {
	var output strings.Builder
	textForCodePoint := func(cp int32) []uint16 {
		if cp <= 0xffff {
			return []uint16{uint16(cp)}
		}
		cp -= 0x10000
		return []uint16{uint16(0xd800 + (cp >> 10)), uint16(0xdc00 + (cp & 0x3ff))}
	}
	units := func(s *JavaString) string {
		var out strings.Builder
		for _, unit := range s.UTF16Copy() {
			fmt.Fprintf(&out, "%x,", unit)
		}
		return out.String()
	}
	literal := JavaStringLiteralUTF16(nil)
	check := func(label string, input []uint16) {
		s, other := NewJavaStringUTF16(input), NewJavaStringUTF16(input)
		trimmed, stripped := JavaStringTrim(s), JavaStringStrip(s)
		fmt.Fprintf(&output, "%s|%s|%t|%s|%s|%t,%t,%t,%t,%t,%t,%t,%t|%s\n",
			label, units(s), JavaStringIsBlank(s), units(trimmed), units(stripped),
			trimmed == s, stripped == s, trimmed == literal, stripped == literal,
			trimmed == JavaStringTrim(s), stripped == JavaStringStrip(s),
			trimmed == JavaStringTrim(other), stripped == JavaStringStrip(other), units(s))
	}
	exception := func(label string, action func()) {
		defer func() {
			err := recover()
			if err == nil {
				fmt.Fprintf(&output, "%s=NO_EXCEPTION\n", label)
				return
			}
			typed, ok := err.(interface{ ThrowableTypeName() string })
			if !ok {
				t.Fatalf("%s non-Java panic %T:%v", label, err, err)
			}
			fmt.Fprintf(&output, "%s=%s\n", label, typed.ThrowableTypeName())
		}()
		action()
	}
	boundary := func(cp int32) {
		middle := textForCodePoint(cp)
		check(fmt.Sprintf("single.%d", cp), middle)
		padded := []uint16{' ', 0x2003}
		padded = append(padded, middle...)
		padded = append(padded, 0x2003, ' ')
		check(fmt.Sprintf("padded.%d", cp), padded)
		ends := append([]uint16{}, middle...)
		ends = append(ends, 'X')
		ends = append(ends, middle...)
		check(fmt.Sprintf("ends.%d", cp), ends)
	}
	output.WriteString("whitespace=")
	for cp := int32(0); cp <= 0x10ffff; cp++ {
		if JavaStringIsBlank(NewJavaStringUTF16(textForCodePoint(cp))) {
			fmt.Fprintf(&output, "%x,", cp)
		}
	}
	output.WriteByte('\n')
	check("empty", nil)
	check("plain", []uint16{'a', 'b', 'c'})
	check("all.trim.controls", []uint16{0, 1, 8, 9, 10, 13, 27, 28, 31, 32})
	check("all.strip", []uint16{9, 10, 11, 12, 13, 28, 29, 30, 31, 32, 0x1680, 0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, 0x2006, 0x2008, 0x2009, 0x200a, 0x2028, 0x2029, 0x205f, 0x3000})
	check("interior", []uint16{' ', 0x2003, 'A', ' ', 0x2003, 'B', 0x2003, ' '})
	check("isolated.forward", []uint16{' ', 0xd800, ' ', 0xdc00, ' '})
	check("isolated.reverse", []uint16{0x2003, 0xdc00, ' ', 0xd800, 0x2003})
	check("surrogate.pair", []uint16{0x2003, 0xd83d, 0xde00, 0x2003})
	check("nonbreaking", []uint16{0xa0, ' ', 0x2007, ' ', 0x202f})
	for cp := int32(0); cp <= 0xa1; cp++ {
		boundary(cp)
	}
	for _, center := range []int32{0x1680, 0x180e, 0x2000, 0x2003, 0x2005, 0x2007, 0x200a, 0x2028, 0x2029, 0x202f, 0x205f, 0x2060, 0x3000, 0xd800, 0xdc00, 0xfeff, 0xfffe, 0x10000, 0x1f600, 0x10fffe} {
		for delta := int32(-1); delta <= 1; delta++ {
			boundary(center + delta)
		}
	}
	exception("null.trim", func() { JavaStringTrim(nil) })
	exception("null.strip", func() { JavaStringStrip(nil) })
	exception("null.blank", func() { JavaStringIsBlank(nil) })
	oracle, err := os.ReadFile("testdata/string_whitespace_jdk21.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, want := strings.Split(output.String(), "\n"), strings.Split(string(oracle), "\n")
	if len(got) != len(want) {
		t.Fatalf("line count: Go %d; JDK21 %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d\nGo: %s\nJDK21: %s", i+1, got[i], want[i])
		}
	}
}
