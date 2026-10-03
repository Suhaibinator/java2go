package stdjava

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf16"
)

// This file implements the subset of java.lang.String behavior that differs
// from Go's native string handling. Java strings are sequences of UTF-16 code
// units, while Go strings are UTF-8 byte sequences. Observation helpers decode
// UTF-16 code units, including surrogate pairs for supplementary characters.
// The string storage remains UTF-8; operations that can produce unpaired
// surrogates still require a representation that preserves those units.
//
// Documented approximations (Java semantics not fully reproduced):
//   - Substring still indexes by rune, so it cannot split surrogate pairs.
//   - StringSplit: Java's regex flavor (java.util.regex) is approximated by Go's
//     RE2 (regexp); patterns using Java-only constructs (backreferences,
//     possessive quantifiers, lookaround) are rejected explicitly rather than
//     silently treated as literal separators.

// regexMetacharacters reports whether the pattern contains any character that
// Java's String.split would interpret as a regex operator. A pattern with none
// is a plain literal and can be split with strings.Split (faster, and exact).
func regexMetacharacters(pattern string) bool {
	return strings.ContainsAny(pattern, `\.[]{}()*+?^$|`)
}

// StringSplit splits around regex matches with Java's optional limit. A
// positive limit bounds the number of fields; zero removes trailing empty
// fields; negative limits retain them. Unsupported regex syntax fails explicitly.
func StringSplit(s, pattern string, limits ...int32) []string {
	limit := int32(0)
	if len(limits) != 0 {
		limit = limits[0]
	}
	count := -1
	if limit > 0 {
		count = int(limit)
	}
	var parts []string
	if !regexMetacharacters(pattern) {
		parts = strings.SplitN(s, pattern, count)
	} else {
		re, err := regexp.Compile(pattern)
		if err != nil {
			panic(NewUnsupportedOperationException("unsupported Java regular expression: " + err.Error()))
		}
		parts = re.Split(s, count)
	}
	// Empty input with no consuming match produces one empty field in Java.
	if s == "" {
		return []string{""}
	}
	if limit == 0 {
		end := len(parts)
		for end > 0 && parts[end-1] == "" {
			end--
		}
		parts = parts[:end]
	}
	return parts
}

// StringSplitArray is the generated Java-array ABI for String.split. The
// slice-returning StringSplit remains available to runtime callers, while
// transpiled code retains String[] identity, covariance, and cast behavior.
func StringSplitArray(s, pattern string, limits ...int32) *ReferenceArray {
	parts := StringSplit(s, pattern, limits...)
	elements := make([]any, len(parts))
	for index, part := range parts {
		elements[index] = part
	}
	return ReferenceArrayLiteral(StringTypeID, elements...)
}

// StringCharAt returns the UTF-16 code unit at index, matching Java's charAt.
// Java char values use Go rune storage, including individual surrogate values.
func StringCharAt(s string, index int32) rune {
	if index >= 0 {
		remaining := index
		for _, value := range s {
			if value > 0xffff {
				high, low := utf16.EncodeRune(value)
				if remaining == 0 {
					return high
				}
				if remaining == 1 {
					return low
				}
				remaining -= 2
			} else {
				if remaining == 0 {
					return value
				}
				remaining--
			}
		}
	}
	// This is a String index contract, not an array access. Preserve the Java
	// subtype and report the UTF-16 length even for negative or extreme indices.
	panic(NewStringIndexOutOfBoundsException(fmt.Sprintf("Index %d out of bounds for length %d", index, StringLength(s))))
}

// StringLength returns the number of UTF-16 code units in the string.
func StringLength(s string) int32 {
	var length int32
	for _, r := range s {
		length++
		if r > 0xffff {
			length++
		}
	}
	return length
}

// StringSubstring returns the substring starting at beginIndex (rune-based),
// matching Java's String.substring(int). Java indexes by UTF-16 code unit; we
// index by rune, which agrees for BMP characters.
func StringSubstring(s string, beginIndex int32) string {
	return string([]rune(s)[beginIndex:])
}

// StringSubstringRange returns the substring in [beginIndex, endIndex)
// (rune-based), matching Java's String.substring(int, int).
func StringSubstringRange(s string, beginIndex, endIndex int32) string {
	return string([]rune(s)[beginIndex:endIndex])
}

// StringEqualsIgnoreCase reports whether s and other are equal ignoring case,
// matching Java's String.equalsIgnoreCase.
func StringEqualsIgnoreCase(s, other string) bool {
	return strings.EqualFold(s, other)
}

// StringCompareTo returns the difference between the first unequal UTF-16
// code units, or the length difference when one string is a prefix of the other.
func StringCompareTo(s, other string) int32 {
	left, right := StringChars(s), StringChars(other)
	for i := 0; i < len(left) && i < len(right); i++ {
		if left[i] != right[i] {
			return left[i] - right[i]
		}
	}
	return int32(len(left) - len(right))
}

// StringReplace replaces all occurrences of old with new, matching Java's
// String.replace(CharSequence, CharSequence).
func StringReplace(s, old, replacement string) string {
	return strings.ReplaceAll(s, old, replacement)
}

// StringIsBlank reports whether the string is empty or contains only whitespace,
// matching Java's String.isBlank (Java 11+).
func StringIsBlank(s string) bool {
	for _, c := range StringRequireNonNull(s) {
		if !CharIsWhitespace(c) {
			return false
		}
	}
	return true
}

// StringTrim removes only characters at or below U+0020, as String.trim does.
func StringTrim(s string) string {
	StringRequireNonNull(s)
	begin, end := 0, len(s)
	for begin < end && s[begin] <= 0x20 {
		begin++
	}
	for end > begin && s[end-1] <= 0x20 {
		end--
	}
	return s[begin:end]
}

// StringStrip uses Java's whitespace predicate, which excludes nonbreaking spaces.
func StringStrip(s string) string {
	return strings.TrimFunc(StringRequireNonNull(s), CharIsWhitespace)
}

// StringChars returns UTF-16 code units stored as runes, matching String.chars.
// Supplementary characters yield separate high and low surrogate values.
func StringChars(s string) []rune {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r > 0xffff {
			high, low := utf16.EncodeRune(r)
			out = append(out, high, low)
		} else {
			out = append(out, r)
		}
	}
	return out
}

// CharIsDigit reports whether the rune is a digit, matching Character.isDigit.
func CharIsDigit(c rune) bool {
	return unicode.IsDigit(c)
}

// CharIsLetter reports whether the rune is a letter, matching Character.isLetter.
func CharIsLetter(c rune) bool {
	return unicode.IsLetter(c)
}

// CharIsLetterOrDigit reports whether the rune is a letter or digit, matching
// Character.isLetterOrDigit.
func CharIsLetterOrDigit(c rune) bool {
	return unicode.IsLetter(c) || unicode.IsDigit(c)
}

// CharIsWhitespace reports whether the rune is whitespace, matching
// Character.isWhitespace.
func CharIsWhitespace(c rune) bool {
	// Java includes the four information separators and excludes nonbreaking
	// spaces U+00A0/U+2007/U+202F and the next-line control U+0085.
	switch c {
	case '\t', '\n', '\v', '\f', '\r', 0x1c, 0x1d, 0x1e, 0x1f, ' ',
		0x1680, 0x2028, 0x2029, 0x205f, 0x3000:
		return true
	}
	return c >= 0x2000 && c <= 0x200a && c != 0x2007
}

// CharIsUpperCase reports whether the rune is uppercase, matching
// Character.isUpperCase.
func CharIsUpperCase(c rune) bool {
	return unicode.IsUpper(c)
}

// CharIsLowerCase reports whether the rune is lowercase, matching
// Character.isLowerCase.
func CharIsLowerCase(c rune) bool {
	return unicode.IsLower(c)
}

// CharToUpperCase returns the uppercase form of the rune, matching
// Character.toUpperCase.
func CharToUpperCase(c rune) rune {
	return unicode.ToUpper(c)
}

// CharToLowerCase returns the lowercase form of the rune, matching
// Character.toLowerCase.
func CharToLowerCase(c rune) rune {
	return unicode.ToLower(c)
}
