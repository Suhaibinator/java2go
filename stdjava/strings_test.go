package stdjava

import "testing"

func TestStringCharAt(t *testing.T) {
	if got := StringCharAt("héllo", 1); got != 'é' {
		t.Errorf("StringCharAt = %q, want é", got)
	}
}

func TestStringLength(t *testing.T) {
	// "héllo" is 6 UTF-8 bytes but 5 characters; Java counts characters.
	if got := StringLength("héllo"); got != 5 {
		t.Errorf("StringLength = %d, want 5", got)
	}
}

func TestStringSubstring(t *testing.T) {
	if got := StringSubstring("héllo", 1); got != "éllo" {
		t.Errorf("StringSubstring = %q, want éllo", got)
	}
	if got := StringSubstringRange("héllo", 1, 3); got != "él" {
		t.Errorf("StringSubstringRange = %q, want él", got)
	}
}

func TestStringIndexOf(t *testing.T) {
	// The 'X' sits after a two-byte rune, so a byte index would be 3 while the
	// rune index Java reports is 2.
	if got := StringIndexOf("éX", "X"); got != 1 {
		t.Errorf("StringIndexOf = %d, want 1", got)
	}
	if got := StringIndexOf("abc", "z"); got != -1 {
		t.Errorf("StringIndexOf(missing) = %d, want -1", got)
	}
	if got := StringLastIndexOf("aXaX", "X"); got != 3 {
		t.Errorf("StringLastIndexOf = %d, want 3", got)
	}
}

func TestStringCompareAndEquals(t *testing.T) {
	if got := StringCompareTo("a", "b"); got >= 0 {
		t.Errorf("StringCompareTo(a,b) = %d, want negative", got)
	}
	if !StringEqualsIgnoreCase("ABC", "abc") {
		t.Errorf("StringEqualsIgnoreCase = false, want true")
	}
}

func TestStringIsBlank(t *testing.T) {
	if !StringIsBlank("  \t ") {
		t.Errorf("StringIsBlank(whitespace) = false, want true")
	}
	if StringIsBlank(" x ") {
		t.Errorf("StringIsBlank(non-blank) = true, want false")
	}
}

func TestCharPredicates(t *testing.T) {
	if !CharIsDigit('7') || CharIsDigit('a') {
		t.Errorf("CharIsDigit misbehaved")
	}
	if !CharIsLetter('a') || CharIsLetter('7') {
		t.Errorf("CharIsLetter misbehaved")
	}
	if CharToUpperCase('a') != 'A' || CharToLowerCase('A') != 'a' {
		t.Errorf("Char case conversion misbehaved")
	}
}
