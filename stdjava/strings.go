package stdjava

import (
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
//   - StringTrim/strip: trim removes chars <= U+0020 in Java, while TrimSpace is
//     Unicode-whitespace aware — a close but not identical approximation.
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
	return StringChars(s)[index]
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

// StringIndexOf returns the UTF-16 index of the first occurrence of substr, or
// -1 if not present, matching Java's String.indexOf.
func StringIndexOf(s, substr string) int32 {
	byteIdx := strings.Index(s, substr)
	if byteIdx < 0 {
		return -1
	}
	return StringLength(s[:byteIdx])
}

// StringLastIndexOf returns the UTF-16 index of the last occurrence of substr,
// or -1 if not present, matching Java's String.lastIndexOf.
func StringLastIndexOf(s, substr string) int32 {
	byteIdx := strings.LastIndex(s, substr)
	if byteIdx < 0 {
		return -1
	}
	return StringLength(s[:byteIdx])
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
	return strings.TrimSpace(s) == ""
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
	return unicode.IsSpace(c)
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
