package stdjava

import (
	"strings"
	"unicode"
)

// This file implements the subset of java.lang.String behavior that differs
// from Go's native string handling. Java strings are sequences of UTF-16 code
// units, while Go strings are UTF-8 byte sequences. For the common case of
// strings within the Basic Multilingual Plane these helpers index by rune,
// which matches Java's char-based indexing for those code points. Characters
// outside the BMP (which Java represents as surrogate pairs) are not modeled;
// the approximation is documented per-function where it matters.

// StringCharAt returns the character at the given index, matching Java's
// String.charAt which returns a char. We model a Java char as a rune. Indexing
// is by rune position rather than byte offset so that multi-byte characters are
// counted as a single position, matching Java for BMP code points.
func StringCharAt(s string, index int32) rune {
	runes := []rune(s)
	return runes[index]
}

// StringLength returns the number of characters in the string. Java counts
// UTF-16 code units; this counts runes, which agrees for BMP characters.
func StringLength(s string) int32 {
	return int32(len([]rune(s)))
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

// StringIndexOf returns the rune index of the first occurrence of substr, or -1
// if not present, matching Java's String.indexOf. The byte offset returned by
// strings.Index is converted to a rune index so the result matches Java for BMP
// characters.
func StringIndexOf(s, substr string) int32 {
	byteIdx := strings.Index(s, substr)
	if byteIdx < 0 {
		return -1
	}
	return int32(len([]rune(s[:byteIdx])))
}

// StringLastIndexOf returns the rune index of the last occurrence of substr, or
// -1 if not present, matching Java's String.lastIndexOf.
func StringLastIndexOf(s, substr string) int32 {
	byteIdx := strings.LastIndex(s, substr)
	if byteIdx < 0 {
		return -1
	}
	return int32(len([]rune(s[:byteIdx])))
}

// StringEqualsIgnoreCase reports whether s and other are equal ignoring case,
// matching Java's String.equalsIgnoreCase.
func StringEqualsIgnoreCase(s, other string) bool {
	return strings.EqualFold(s, other)
}

// StringCompareTo lexicographically compares two strings, matching the sign
// contract of Java's String.compareTo (negative, zero, or positive).
func StringCompareTo(s, other string) int32 {
	return int32(strings.Compare(s, other))
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

// StringChars returns the characters of the string as a slice of runes. Java's
// String.chars returns an IntStream of UTF-16 code units; this returns the runes
// instead, which agrees for BMP characters.
func StringChars(s string) []rune {
	return []rune(s)
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
