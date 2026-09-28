package stdjava

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf16"
)

func parseBigIntegerDecimal(text string) big.Int {
	ReferenceRequireNonNull(text)
	chars := StringChars(text)
	negative := false
	start := 0
	if len(chars) == 0 {
		panic(NewNumberFormatException("Zero length BigInteger"))
	}
	minus, plus := strings.LastIndex(text, "-"), strings.LastIndex(text, "+")
	if minus >= 0 {
		if minus != 0 || plus >= 0 {
			panic(NewNumberFormatException("Illegal embedded sign character"))
		}
		negative = true
		start = 1
	} else if plus >= 0 {
		if plus != 0 {
			panic(NewNumberFormatException("Illegal embedded sign character"))
		}
		start = 1
	}
	if start == len(chars) {
		panic(NewNumberFormatException("Zero length BigInteger"))
	}
	for start < len(chars) && CharDigit(chars[start], 10) == 0 {
		start++
	}
	if start == len(chars) {
		return big.Int{}
	}
	digits := make([]byte, 0, len(chars)-start)
	// JDK parses decimal groups through Integer.parseInt; retain which group
	// fails so malformed long inputs expose the same NumberFormatException text.
	width := (len(chars) - start) % 9
	if width == 0 {
		width = 9
	}
	for start < len(chars) {
		end := start + width
		for _, c := range chars[start:end] {
			digit := CharDigit(c, 10)
			if digit < 0 {
				units := make([]uint16, end-start)
				for i, r := range chars[start:end] {
					units[i] = uint16(r)
				}
				panic(NewNumberFormatException("For input string: \"" + string(utf16.Decode(units)) + "\""))
			}
			digits = append(digits, byte('0'+digit))
		}
		start = end
		width = 9
	}
	var value big.Int
	value.SetString(string(digits), 10)
	if negative {
		value.Neg(&value)
	}
	return value
}

func bigDecimalMissingCharacter(index, length int) {
	failure := NewNumberFormatException(NullString())
	cause := NewArrayIndexOutOfBoundsException("Index " + strconv.Itoa(index) + " out of bounds for length " + strconv.Itoa(length))
	// The freshly allocated exception has not escaped. Initialize its cause
	// directly rather than allocate a logical thread and monitor for construction.
	failure.state.cause = cause
	failure.state.causeInitialized = true
	panic(failure)
}
func parseBigDecimal(text string) (big.Int, int32) {
	ReferenceRequireNonNull(text)
	chars := StringChars(text)
	if len(chars) == 0 {
		bigDecimalMissingCharacter(0, 0)
	}
	start := 0
	negative := false
	if chars[0] == '-' || chars[0] == '+' {
		negative = chars[0] == '-'
		start++
	}
	compact := len(chars)-start <= 18
	dot := false
	digits := make([]byte, 0, len(chars))
	var scale int64
	for i := start; i < len(chars); i++ {
		c := chars[i]
		digit := CharDigit(c, 10)
		if digit >= 0 {
			digits = append(digits, byte('0'+digit))
			if dot {
				scale++
			}
			continue
		}
		if c == '.' {
			if dot {
				panic(NewNumberFormatException("Character array contains more than one decimal point."))
			}
			dot = true
			continue
		}
		if c == 'e' || c == 'E' {
			scale -= parseBigDecimalExponent(chars, i+1)
			break
		}
		if compact {
			panic(NewNumberFormatException("Character " + string(c) + " is neither a decimal digit number, decimal point, nor \"e\" notation exponential mark."))
		}
		panic(NewNumberFormatException("Character array is missing \"e\" notation exponential mark."))
	}
	if len(digits) == 0 {
		panic(NewNumberFormatException("No digits found."))
	}
	if scale < math.MinInt32 || scale > math.MaxInt32 {
		panic(NewNumberFormatException("Exponent overflow."))
	}
	var coefficient big.Int
	coefficient.SetString(string(digits), 10)
	if negative {
		coefficient.Neg(&coefficient)
	}
	return coefficient, int32(scale)
}
func parseBigDecimalExponent(chars []rune, start int) int64 {
	if start >= len(chars) {
		bigDecimalMissingCharacter(start, len(chars))
	}
	negative := false
	if chars[start] == '-' || chars[start] == '+' {
		negative = chars[start] == '-'
		start++
		if start >= len(chars) {
			bigDecimalMissingCharacter(start, len(chars))
		}
	}
	for len(chars)-start > 10 && CharDigit(chars[start], 10) == 0 {
		start++
	}
	if len(chars)-start > 10 {
		panic(NewNumberFormatException("Too many nonzero exponent digits."))
	}
	var exponent int64
	for _, c := range chars[start:] {
		digit := CharDigit(c, 10)
		if digit < 0 {
			panic(NewNumberFormatException("Not a digit."))
		}
		exponent = exponent*10 + int64(digit)
	}
	if negative {
		return -exponent
	}
	return exponent
}
