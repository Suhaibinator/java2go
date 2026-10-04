package stdjava

import "fmt"

// JavaStringIndexOf implements the String and int overloads, including JDK21's
// bounded search. Every offset and match is measured in immutable UTF16 units.
func JavaStringIndexOf(text *JavaString, needle any, bounds ...int32) int32 {
	RequireJavaString(text)
	var pointUnits [2]uint16
	target, valid := javaStringSearchTarget(needle, &pointUnits)
	start, end := int32(0), text.Length()
	switch len(bounds) {
	case 0:
	case 1:
		start = bounds[0]
	case 2:
		start, end = bounds[0], bounds[1]
		if start < 0 || end > text.Length() || start > end {
			panic(NewStringIndexOutOfBoundsException(fmt.Sprintf("Range [%d, %d) out of bounds for length %d", start, end, text.Length())))
		}
	default:
		panic(NewIllegalArgumentException("String.indexOf requires at most two bounds"))
	}
	if !valid {
		return -1
	}
	if start < 0 {
		start = 0
	}
	if start > end {
		if len(target) == 0 {
			return end
		}
		return -1
	}
	for index := int(start); index <= int(end)-len(target); index++ {
		if javaStringUnitsMatchAt(text.units, target, index) {
			return int32(index)
		}
	}
	return -1
}

// JavaStringLastIndexOf uses an inclusive starting offset. A supplementary
// code point matches its surrogate pair and returns the leading-unit index.
func JavaStringLastIndexOf(text *JavaString, needle any, from ...int32) int32 {
	RequireJavaString(text)
	var pointUnits [2]uint16
	target, valid := javaStringSearchTarget(needle, &pointUnits)
	start := text.Length()
	if len(from) > 1 {
		panic(NewIllegalArgumentException("String.lastIndexOf requires at most one starting offset"))
	}
	if len(from) == 1 {
		start = from[0]
	}
	if !valid || start < 0 {
		return -1
	}
	last := len(text.units) - len(target)
	if int64(start) < int64(last) {
		last = int(start)
	}
	for index := last; index >= 0; index-- {
		if javaStringUnitsMatchAt(text.units, target, index) {
			return int32(index)
		}
	}
	return -1
}

// String needles retain their canonical payload without copying. Numeric
// needles borrow caller storage instead of allocating a target slice.
func javaStringSearchTarget(needle any, pointUnits *[2]uint16) ([]uint16, bool) {
	if text, ok := needle.(*JavaString); ok {
		RequireJavaString(text)
		return text.units, true
	}
	point := stringSearchInt(needle)
	if point < 0 || point > 0x10ffff {
		return nil, false
	}
	if point <= 0xffff {
		pointUnits[0] = uint16(point)
		return pointUnits[:1], true
	}
	point -= 0x10000
	pointUnits[0] = uint16(0xd800 + (point >> 10))
	pointUnits[1] = uint16(0xdc00 + (point & 0x3ff))
	return pointUnits[:], true
}
