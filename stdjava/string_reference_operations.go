package stdjava

import "fmt"

// Substring uses UTF16 indexes. JDK21 String.checkBoundsBeginEnd delegates to
// Preconditions.checkFromToIndex, whose exception formatter produces this text.
func (s *JavaString) Substring(begin, end int32) *JavaString {
	ReferenceRequireNonNull(s)
	length := s.Length()
	if begin < 0 || end > length || begin > end {
		panic(NewStringIndexOutOfBoundsException(fmt.Sprintf("Range [%d, %d) out of bounds for length %d", begin, end, length)))
	}
	if begin == 0 && end == length {
		return s
	}
	// StringLatin1.newString and StringUTF16.newString return the empty
	// literal for an empty proper substring. A full empty range returned above
	// must preserve even a freshly allocated empty receiver.
	if begin == end {
		return JavaStringLiteralUTF16(nil)
	}
	return NewJavaStringUTF16(s.units[begin:end])
}

// Concat implements String.concat, whose empty argument preserves its receiver.
func (s *JavaString) Concat(other *JavaString) *JavaString {
	ReferenceRequireNonNull(s)
	ReferenceRequireNonNull(other)
	if other.Length() == 0 {
		return s
	}
	return ConcatJavaStrings(s, other)
}

// ConcatJavaStrings joins already-converted nonnull operands of runtime String
// addition. Even an empty operand produces a new wrapper (JLS15.18.1 and JDK21
// StringConcatHelper.simpleConcat). Compiler lowering owns source evaluation
// order and null/primitive/object conversion before invoking this helper.
func ConcatJavaStrings(left, right *JavaString) *JavaString {
	ReferenceRequireNonNull(left)
	ReferenceRequireNonNull(right)
	if left.Length() == 0 {
		return CopyJavaString(right)
	}
	if right.Length() == 0 {
		return CopyJavaString(left)
	}
	units := make([]uint16, len(left.units)+len(right.units))
	copy(units, left.units)
	copy(units[len(left.units):], right.units)
	// units is freshly owned and never exposed; avoid a redundant defensive copy.
	return &JavaString{units: units}
}
