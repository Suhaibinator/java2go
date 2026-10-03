package stdjava

// JavaStringStartsWith compares UTF16 units at the selected Java offset. JDK21
// checks a negative offset before dereferencing the prefix, so a null prefix
// returns false in that case. The receiver still requires a nonnull reference.
func JavaStringStartsWith(text, prefix *JavaString, offsets ...int32) bool {
	ReferenceRequireNonNull(text)
	var offset int32
	if len(offsets) != 0 {
		offset = offsets[0]
	}
	if offset < 0 {
		return false
	}
	ReferenceRequireNonNull(prefix)
	return javaStringUnitsMatchAt(text.units, prefix.units, int(offset))
}

// JavaStringEndsWith compares the suffix as UTF16 units, retaining isolated
// surrogate units and rejecting an argument longer than the receiver.
func JavaStringEndsWith(text, suffix *JavaString) bool {
	ReferenceRequireNonNull(text)
	ReferenceRequireNonNull(suffix)
	return javaStringUnitsMatchAt(text.units, suffix.units, len(text.units)-len(suffix.units))
}

func javaStringUnitsMatchAt(text, prefix []uint16, offset int) bool {
	if offset < 0 || offset > len(text)-len(prefix) {
		return false
	}
	for i, unit := range prefix {
		if text[offset+i] != unit {
			return false
		}
	}
	return true
}
