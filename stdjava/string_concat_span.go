package stdjava

// checkedConcatUTF16Length admits only the nonnegative int32 representation
// domain. Length() narrows raw lengths to int32; a host-int-only guard would
// change the original binary helper's empty branches outside this domain.
func checkedConcatUTF16Length(total, next int) (int, bool) {
	return checkedJoiningUTF16Length(total, next, 1, int(^uint32(0)>>1))
}

func concatJavaStringSpanFold(first, second *JavaString, tail []*JavaString) *JavaString {
	result := ConcatJavaStrings(first, second)
	for _, part := range tail {
		result = ConcatJavaStrings(result, part)
	}
	return result
}

// ConcatJavaStringSpan joins already converted operands into a fresh dynamic
// String. The compiler admits only scalar, nonconstant String concatenations.
// Rejected representation edges replay the original ordered binary operation;
// no new Java size limit or physical allocation-failure contract is invented.
func ConcatJavaStringSpan(first, second *JavaString, tail ...*JavaString) *JavaString {
	if first == nil || second == nil {
		return concatJavaStringSpanFold(first, second, tail)
	}
	total, ok := checkedConcatUTF16Length(0, len(first.units))
	if !ok {
		return concatJavaStringSpanFold(first, second, tail)
	}
	total, ok = checkedConcatUTF16Length(total, len(second.units))
	if !ok {
		return concatJavaStringSpanFold(first, second, tail)
	}
	for _, part := range tail {
		if part == nil {
			return concatJavaStringSpanFold(first, second, tail)
		}
		total, ok = checkedConcatUTF16Length(total, len(part.units))
		if !ok {
			return concatJavaStringSpanFold(first, second, tail)
		}
	}
	units := make([]uint16, total)
	offset := copy(units, first.units)
	offset += copy(units[offset:], second.units)
	for _, part := range tail {
		offset += copy(units[offset:], part.units)
	}
	return &JavaString{units: units}
}
