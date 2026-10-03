package stdjava

// JavaStringTrim removes leading/trailing units at or below U+0020. The checked
// substring core preserves an unchanged receiver, including a fresh empty one,
// and gives a changed nonempty result its own immutable wrapper and unit copy.
func JavaStringTrim(text *JavaString) *JavaString {
	ReferenceRequireNonNull(text)
	left, right := 0, len(text.units)
	for left < right && text.units[left] <= 0x0020 {
		left++
	}
	for left < right && text.units[right-1] <= 0x0020 {
		right--
	}
	return text.Substring(int32(left), int32(right))
}

// JavaStringStrip uses the existing explicit CharIsWhitespace table, verified
// against every JDK21 codepoint. The table has no supplementary whitespace, so
// scanning UTF16 units also rejects both pairs and isolated surrogate units.
// An empty or all-whitespace receiver
// yields the empty literal, even when the receiver was a fresh empty wrapper.
func JavaStringStrip(text *JavaString) *JavaString {
	ReferenceRequireNonNull(text)
	left, right := 0, len(text.units)
	for left < right && CharIsWhitespace(rune(text.units[left])) {
		left++
	}
	if left == right {
		return JavaStringLiteralUTF16(nil)
	}
	for CharIsWhitespace(rune(text.units[right-1])) {
		right--
	}
	return text.Substring(int32(left), int32(right))
}

// JavaStringIsBlank accepts empty content and strings made only of JDK21
// whitespace, preserving the distinction between trim controls and whitespace.
func JavaStringIsBlank(text *JavaString) bool {
	ReferenceRequireNonNull(text)
	for _, unit := range text.units {
		if !CharIsWhitespace(rune(unit)) {
			return false
		}
	}
	return true
}
