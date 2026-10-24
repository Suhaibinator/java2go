package stdjava

import "fmt"

// JavaStringFromChars copies Java char units into a fresh String wrapper.
// The generated char ABI uses rune, but each value denotes one UTF16 unit.
func JavaStringFromChars(chars *PrimitiveArray[rune]) *JavaString {
	if chars == nil {
		panic(NewNullPointerException(`Cannot read the array length because "value" is null`))
	}
	return JavaStringFromCharsRange(chars, 0, int32(len(chars.Elements)))
}

// JavaStringFromCharsRange implements String(char[], offset, count), checking
// null before bounds and avoiding Java int overflow in the offset/count sum.
func JavaStringFromCharsRange(chars *PrimitiveArray[rune], offset, count int32) *JavaString {
	if chars == nil {
		panic(NewNullPointerException(`Cannot read the array length because "value" is null`))
	}
	length := int64(len(chars.Elements))
	if offset < 0 || count < 0 || int64(offset)+int64(count) > length {
		panic(NewStringIndexOutOfBoundsException(fmt.Sprintf("Range [%d, %d + %d) out of bounds for length %d", offset, offset, count, length)))
	}
	units := make([]uint16, int(count))
	for index := range units {
		units[index] = uint16(chars.Elements[int(offset)+index])
	}
	// The new backing storage is exclusively owned by this fresh wrapper.
	return &JavaString{units: units}
}

// JavaStringToCharArray returns a fresh mutable array containing exact UTF16
// code units; neither isolated surrogates nor pairs are decoded as code points.
func JavaStringToCharArray(text *JavaString) *PrimitiveArray[rune] {
	ReferenceRequireNonNull(text)
	chars := NewPrimitiveArray[rune](len(text.units), PrimitiveTypeID("char"))
	for index, unit := range text.units {
		chars.Elements[index] = rune(unit)
	}
	return chars
}

// JavaStringSubstringFrom shares the checked core's bounds and identity rules.
func JavaStringSubstringFrom(text *JavaString, begin int32) *JavaString {
	return text.Substring(begin, text.Length())
}
