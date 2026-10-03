package stdjava

import (
	"unicode"
	"unicode/utf16"
)

// NewStringBuilderJavaString copies Java content without an encoding boundary.
func NewStringBuilderJavaString(text *JavaString) *StringBuilder {
	RequireJavaString(text)
	return NewStringBuilder().AppendJavaString(text)
}

// AppendJavaString is the String overload: a null reference contributes "null".
func (b *StringBuilder) AppendJavaString(text *JavaString) *StringBuilder {
	ReferenceRequireNonNull(b)
	if text == nil {
		text = JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'})
	}
	for _, unit := range text.units {
		b.buf = append(b.buf, rune(unit))
	}
	return b
}

func (b *StringBuilder) InsertJavaString(offset int32, text *JavaString) *StringBuilder {
	ReferenceRequireNonNull(b)
	b.checkOffset(offset)
	if text == nil {
		text = JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'})
	}
	inserted := make([]rune, len(text.units))
	for i, unit := range text.units {
		inserted[i] = rune(unit)
	}
	return b.Insert(offset, inserted)
}

// JavaStringJoinIterableExecution retains iterator/checkcast and conversion
// ordering from String.join; source toString results are never encoded.
func JavaStringJoinIterableExecution(execution *Execution, delimiter, elements any) *JavaString {
	requireExecution(execution)
	ReferenceRequireNonNull(delimiter)
	ReferenceRequireNonNull(elements)
	separator := JavaStringValueOfExecution(execution, delimiter)
	iterable, ok := elements.(JavaIterable)
	if !ok {
		panic(NewUnsupportedOperationException("Iterable iterator protocol is not implemented"))
	}
	iterator := IterableIteratorExecution(execution, iterable)
	var parts []*JavaString
	for IteratorHasNextExecution(execution, iterator) {
		element := stringJoinCharSequence(IteratorNextExecution(execution, iterator))
		parts = append(parts, JavaStringValueOfExecution(execution, element))
	}
	return javaStringJoinConverted(separator, parts)
}

func JavaStringJoinArrayExecution(execution *Execution, delimiter any, elements *ReferenceArray) *JavaString {
	requireExecution(execution)
	ReferenceRequireNonNull(delimiter)
	separator := JavaStringValueOfExecution(execution, delimiter)
	length := ReferenceArrayLength(elements)
	parts := make([]*JavaString, 0, length)
	for index := int32(0); index < length; index++ {
		element := ReferenceArrayGet[any](elements, index, CharSequenceTypeID)
		parts = append(parts, JavaStringValueOfExecution(execution, element))
	}
	return javaStringJoinConverted(separator, parts)
}

func JavaStringJoinValuesExecution(execution *Execution, delimiter any, elements ...any) *JavaString {
	requireExecution(execution)
	ReferenceRequireNonNull(delimiter)
	separator := JavaStringValueOfExecution(execution, delimiter)
	parts := make([]*JavaString, 0, len(elements))
	for _, element := range elements {
		parts = append(parts, JavaStringValueOfExecution(execution, stringJoinCharSequence(element)))
	}
	return javaStringJoinConverted(separator, parts)
}

func javaStringJoinConverted(separator *JavaString, parts []*JavaString) *JavaString {
	if len(parts) > 1 {
		RequireJavaString(separator)
	}
	for _, part := range parts {
		RequireJavaString(part)
	}
	var units []uint16
	for index, part := range parts {
		if index > 0 {
			units = append(units, separator.units...)
		}
		units = append(units, part.units...)
	}
	return &JavaString{units: units}
}

// JavaStringStreamJoining is the canonical String-element collector adapter.
func JavaStringStreamJoining(stream Stream[*JavaString], separator, prefix, suffix *JavaString) *JavaString {
	RequireJavaString(separator)
	RequireJavaString(prefix)
	RequireJavaString(suffix)
	// Size concrete immutable strings without re-running stream callbacks.
	// Decline unsafe arithmetic and retain the original append path below.
	limit := int(^uint(0)>>1) / 2
	if limit > 2147483647 {
		limit = 2147483647
	}
	total, planned := checkedJoiningUTF16Length(0, len(prefix.units), 1, limit)
	if planned {
		total, planned = checkedJoiningUTF16Length(total, len(suffix.units), 1, limit)
	}
	gaps := len(stream.elements)
	if gaps > 0 {
		gaps--
	}
	if planned {
		total, planned = checkedJoiningUTF16Length(total, len(separator.units), gaps, limit)
	}
	if planned {
		for _, element := range stream.elements {
			length := 4 // The original append loop emits "null" for nil elements.
			if element != nil {
				length = len(element.units)
			}
			total, planned = checkedJoiningUTF16Length(total, length, 1, limit)
			if !planned {
				break
			}
		}
	}
	var units []uint16
	if planned {
		if total > 0 {
			units = make([]uint16, 0, total)
		}
		units = append(units, prefix.units...)
	} else {
		units = append([]uint16(nil), prefix.units...)
	}
	for index, element := range stream.elements {
		if index > 0 {
			units = append(units, separator.units...)
		}
		if element == nil {
			units = append(units, 'n', 'u', 'l', 'l')
		} else {
			units = append(units, element.units...)
		}
	}
	units = append(units, suffix.units...)
	return &JavaString{units: units}
}

// checkedJoiningUTF16Length admits total + length*count only when every
// nonnegative quantity and the resulting UTF-16 size fit the supplied limit.
func checkedJoiningUTF16Length(total, length, count, limit int) (int, bool) {
	if total < 0 || length < 0 || count < 0 || limit < 0 || total > limit {
		return 0, false
	}
	if length == 0 || count == 0 {
		return total, true
	}
	if count > (limit-total)/length {
		return 0, false
	}
	return total + length*count, true
}

// JavaStringRegionMatches compares UTF16 ranges, decoding only valid pairs for
// case folding. Isolated units remain unchanged and participate in comparison.
func JavaStringRegionMatches(value *JavaString, ignoreCase bool, offset int32, other *JavaString, otherOffset, length int32) bool {
	RequireJavaString(value)
	RequireJavaString(other)
	left, right := value.units, other.units
	if offset < 0 || otherOffset < 0 || int64(offset) > int64(len(left))-int64(length) || int64(otherOffset) > int64(len(right))-int64(length) {
		return false
	}
	for index := int32(0); index < length; {
		a, b := rune(left[offset+index]), rune(right[otherOffset+index])
		width := int32(1)
		if ignoreCase && index+1 < length && a >= 0xd800 && a <= 0xdbff && b >= 0xd800 && b <= 0xdbff {
			al, bl := rune(left[offset+index+1]), rune(right[otherOffset+index+1])
			if al >= 0xdc00 && al <= 0xdfff && bl >= 0xdc00 && bl <= 0xdfff {
				a, b = utf16.DecodeRune(a, al), utf16.DecodeRune(b, bl)
				width = 2
			}
		}
		if a != b {
			if !ignoreCase {
				return false
			}
			a, b = unicode.ToUpper(a), unicode.ToUpper(b)
			if a != b && unicode.ToLower(a) != unicode.ToLower(b) {
				return false
			}
		}
		index += width
	}
	return true
}
