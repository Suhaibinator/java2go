package stdjava

import "unicode/utf16"

// javaUTF8SignedDecodeUnit reuses the streaming decoder's Java malformed-input
// policy without copying the whole mutable byte array. Four bytes cover every
// possible UTF8 unit, so a full window cannot be an incomplete sequence.
func javaUTF8SignedDecodeUnit(input []int8) (rune, int) {
	var window [4]byte
	count := min(len(input), len(window))
	for index := 0; index < count; index++ {
		window[index] = byte(input[index])
	}
	point, consumed, _ := javaUTF8Unit(window[:count], true)
	return point, consumed
}

// javaUTF8OwnedDecodeLength checks the final UTF16 size before allocation.
// Refusing a conservative bound leaves the existing fallback in charge.
func javaUTF8OwnedDecodeLength(input []int8, limit int) (int, bool) {
	if limit < 0 {
		return 0, false
	}
	length := 0
	for index := 0; index < len(input); {
		point, consumed := javaUTF8SignedDecodeUnit(input[index:])
		width := 1
		if point > 0xffff {
			width = 2
		}
		if width > limit-length {
			return 0, false
		}
		length += width
		index += consumed
	}
	return length, true
}

// decodeJavaUTF8OwnedSigned fills exclusively new UTF16 storage and returns a
// fresh, noninterned Java wrapper, including when the decoded content is empty.
func decodeJavaUTF8OwnedSigned(input []int8) (*JavaString, bool) {
	limit := int(^uint(0) >> 1)
	if limit > 2147483647 {
		limit = 2147483647
	}
	length, ok := javaUTF8OwnedDecodeLength(input, limit)
	if !ok {
		return nil, false
	}
	units := make([]uint16, length)
	offset := 0
	for index := 0; index < len(input); {
		point, consumed := javaUTF8SignedDecodeUnit(input[index:])
		width := 1
		if point > 0xffff {
			width = 2
		}
		// The Java byte array is mutable. Never turn a changed count into a host
		// bounds panic; conservatively retry through the existing copying fallback.
		if width > len(units)-offset {
			return nil, false
		}
		if width == 2 {
			high, low := utf16.EncodeRune(point)
			units[offset], units[offset+1] = uint16(high), uint16(low)
		} else {
			units[offset] = uint16(point)
		}
		offset += width
		index += consumed
	}
	return &JavaString{units: units[:offset]}, true
}
