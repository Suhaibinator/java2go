package transpiler

import (
	"fmt"
	"strconv"
	"strings"
)

// Normalize escapes whose syntax or value differs between Java and Go.
// Java permits escaped double quotes in chars, single quotes in strings, and
// a space escape in either; Go does not. Java octal escapes consume one or two
// digits, or three when the first is
// 0..3. Go requires exactly three octal digits and interprets string octal
// escapes as bytes, whereas Java's values are Unicode characters. Emit a
// Unicode escape for the decoded value, leaving adjacent digits and every
// unrelated escape intact. In particular, an escaped backslash must not start
// a second decoding pass.
func normalizeJavaLiteralEscapes(literal string) string {
	if !strings.ContainsRune(literal, '\\') {
		return literal
	}
	var result strings.Builder
	for index := 0; index < len(literal); index++ {
		if literal[index] != '\\' || index+1 == len(literal) {
			result.WriteByte(literal[index])
			continue
		}
		first := literal[index+1]
		if first == 's' {
			result.WriteByte(' ')
			index++
			continue
		}
		if (literal[0] == '\'' && first == '"') || (literal[0] == '"' && first == '\'') {
			result.WriteByte(first)
			index++
			continue
		}
		if first < '0' || first > '7' {
			result.WriteString(literal[index : index+2])
			index++
			continue
		}
		limit := 2
		if first <= '3' {
			limit = 3
		}
		value := 0
		digits := 0
		for digits < limit && index+1 < len(literal) && literal[index+1] >= '0' && literal[index+1] <= '7' {
			index++
			value = value*8 + int(literal[index]-'0')
			digits++
		}
		fmt.Fprintf(&result, "\\u%04x", value)
	}
	return result.String()
}

// A Java char is a UTF-16 code unit, so an escaped surrogate is a valid char.
// Go rune literals reject those values as Unicode scalar escapes. A numeric
// constant retains the exact unit without making a claim about String storage.
func normalizeJavaCharacterLiteral(literal string) string {
	normalized := normalizeJavaLiteralEscapes(literal)
	if strings.HasPrefix(normalized, "'\\u") && strings.HasSuffix(normalized, "'") {
		digits := strings.TrimLeft(normalized[2:len(normalized)-1], "u")
		if len(digits) == 4 {
			if unit, err := strconv.ParseUint(digits, 16, 16); err == nil && unit >= 0xd800 && unit <= 0xdfff {
				return strconv.FormatUint(unit, 10)
			}
		}
	}
	return normalized
}
