package stdjava

import "unicode/utf16"

func stringSearchInt(value any) int32 {
	ReferenceRequireNonNull(value)
	switch number := value.(type) {
	case int:
		return int32(number)
	case int32:
		return number
	case int16:
		return int32(number)
	case int8:
		return int32(number)
	case *Integer:
		return UnboxInteger(number)
	case *Character:
		return UnboxCharacter(number)
	case *Short:
		return int32(UnboxShort(number))
	case *Byte:
		return int32(UnboxByte(number))
	default:
		panic(NewIllegalArgumentException("String search requires an int index or code point"))
	}
}

func stringSearchUnits(needle any) ([]rune, bool) {
	if text, ok := needle.(string); ok {
		return StringChars(StringRequireNonNull(text)), true
	}
	point := stringSearchInt(needle)
	if point < 0 || point > 0x10ffff {
		return nil, false
	}
	if point <= 0xffff {
		return []rune{point}, true
	}
	high, low := utf16.EncodeRune(point)
	return []rune{high, low}, true
}

// StringIndexOf handles the String and int-code-point overloads. Indexes and
// optional starting offsets count UTF-16 units, including individual surrogates.
func StringIndexOf(s string, needle any, from ...any) int32 {
	units := StringChars(StringRequireNonNull(s))
	start := int32(0)
	if len(from) != 0 {
		start = stringSearchInt(from[0])
	}
	target, valid := stringSearchUnits(needle)
	if !valid {
		return -1
	}
	if start < 0 {
		start = 0
	}
	if int64(start) > int64(len(units)) {
		if len(target) == 0 {
			return int32(len(units))
		}
		return -1
	}
	for index := int(start); index <= len(units)-len(target); index++ {
		if stringSearchMatches(units[index:], target) {
			return int32(index)
		}
	}
	return -1
}

// StringLastIndexOf searches backwards from the inclusive UTF-16 start offset.
func StringLastIndexOf(s string, needle any, from ...any) int32 {
	units := StringChars(StringRequireNonNull(s))
	start := int32(len(units))
	if len(from) != 0 {
		start = stringSearchInt(from[0])
	}
	target, valid := stringSearchUnits(needle)
	if !valid || start < 0 {
		return -1
	}
	last := len(units) - len(target)
	if int64(start) < int64(last) {
		last = int(start)
	}
	for index := last; index >= 0; index-- {
		if stringSearchMatches(units[index:], target) {
			return int32(index)
		}
	}
	return -1
}

func stringSearchMatches(units, target []rune) bool {
	for index, unit := range target {
		if units[index] != unit {
			return false
		}
	}
	return true
}
