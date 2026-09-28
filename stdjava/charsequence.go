package stdjava

import (
	"unicode"
	"unicode/utf16"
)

// CharSequence operations preserve the dynamic implementation's dispatch even
// when the Java interface is represented as any by generated Go.
func CharSequenceLength(execution *Execution, value any) int32 {
	ReferenceRequireNonNull(value)
	switch value := value.(type) {
	case string:
		return StringLength(value)
	case interface{ LengthJava2goExecution(*Execution) int32 }:
		return value.LengthJava2goExecution(execution)
	case interface{ Length() int32 }:
		return value.Length()
	}
	panic(NewClassCastException("value does not implement CharSequence.length"))
}
func CharSequenceCharAt(execution *Execution, value any, index int32) rune {
	ReferenceRequireNonNull(value)
	switch value := value.(type) {
	case string:
		return StringCharAt(value, index)
	case interface{ CharAtJava2goExecution(*Execution, int32) rune }:
		return value.CharAtJava2goExecution(execution, index)
	case interface{ CharAt(int32) rune }:
		return value.CharAt(index)
	}
	panic(NewClassCastException("value does not implement CharSequence.charAt"))
}

// StringRegionMatches uses UTF-16 offsets. Case-insensitive comparison observes
// supplementary case pairs as code points, matching the JDK's UTF-16 path.
func StringRegionMatches(value string, ignoreCase bool, offset int32, other string, otherOffset, length int32) bool {
	StringRequireNonNull(value)
	StringRequireNonNull(other)
	left, right := StringChars(value), StringChars(other)
	if offset < 0 || otherOffset < 0 || int64(offset) > int64(len(left))-int64(length) || int64(otherOffset) > int64(len(right))-int64(length) {
		return false
	}
	for index := int32(0); index < length; {
		a, b := left[offset+index], right[otherOffset+index]
		width := int32(1)
		if ignoreCase && index+1 < length && a >= 0xd800 && a <= 0xdbff && b >= 0xd800 && b <= 0xdbff {
			al, bl := left[offset+index+1], right[otherOffset+index+1]
			if al >= 0xdc00 && al <= 0xdfff && bl >= 0xdc00 && bl <= 0xdfff {
				a, b = utf16.DecodeRune(a, al), utf16.DecodeRune(b, bl)
				width = 2
			}
		}
		if a == b {
			index += width
			continue
		}
		if !ignoreCase {
			return false
		}
		a, b = unicode.ToUpper(a), unicode.ToUpper(b)
		if a != b && unicode.ToLower(a) != unicode.ToLower(b) {
			return false
		}
		index += width
	}
	return true
}
