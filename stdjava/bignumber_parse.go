package stdjava

import (
	"math"
	"math/big"
	"strconv"
	"unicode/utf16"
)

// bigNumberNativeJavaString is the legacy host-text ingress. Canonical Java
// callers enter the shared parsers directly without decoding their UTF16 units.
func bigNumberNativeJavaString(text string) *JavaString {
	ReferenceRequireNonNull(text)
	return NewJavaStringUTF16(utf16.Encode([]rune(text)))
}

// bigNumberJavaString converts runtime-authored numeric output or diagnostics;
// application String payloads must never pass through this host-text boundary.
func bigNumberJavaString(text string) *JavaString {
	return NewJavaStringUTF16(utf16.Encode([]rune(text)))
}

func bigNumberFormatException(message string) NumberFormatException {
	return NewJavaNumberFormatException(bigNumberJavaString(message))
}

func bigNumberDecimalDigit(unit uint16) int32 {
	digit := javaIntegerDigit21(unit)
	if digit >= 10 {
		return -1
	}
	return digit
}

func parseBigIntegerDecimal(text string) big.Int {
	return parseBigIntegerDecimalJavaString(bigNumberNativeJavaString(text))
}

func parseBigIntegerDecimalJavaString(text *JavaString) big.Int {
	ReferenceRequireNonNull(text)
	chars := text.units
	negative := false
	start := 0
	if len(chars) == 0 {
		panic(bigNumberFormatException("Zero length BigInteger"))
	}
	minus, plus := -1, -1
	for i, unit := range chars {
		switch unit {
		case '-':
			minus = i
		case '+':
			plus = i
		}
	}
	if minus >= 0 {
		if minus != 0 || plus >= 0 {
			panic(bigNumberFormatException("Illegal embedded sign character"))
		}
		negative = true
		start = 1
	} else if plus >= 0 {
		if plus != 0 {
			panic(bigNumberFormatException("Illegal embedded sign character"))
		}
		start = 1
	}
	if start == len(chars) {
		panic(bigNumberFormatException("Zero length BigInteger"))
	}
	for start < len(chars) && bigNumberDecimalDigit(chars[start]) == 0 {
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
			digit := bigNumberDecimalDigit(c)
			if digit < 0 {
				panic(javaIntegerParseInputException(&JavaString{units: chars[start:end]}, 10))
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
	failure := NewJavaNumberFormatException(nil)
	cause := NewArrayIndexOutOfBoundsException("Index " + strconv.Itoa(index) + " out of bounds for length " + strconv.Itoa(length))
	// The freshly allocated exception has not escaped. Initialize its cause
	// directly rather than allocate a logical thread and monitor for construction.
	failure.state.cause = cause
	failure.state.causeInitialized = true
	panic(failure)
}
func parseBigDecimal(text string) (big.Int, int32) {
	return parseBigDecimalJavaString(bigNumberNativeJavaString(text))
}

func parseBigDecimalJavaString(text *JavaString) (big.Int, int32) {
	ReferenceRequireNonNull(text)
	chars := text.units
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
		digit := bigNumberDecimalDigit(c)
		if digit >= 0 {
			digits = append(digits, byte('0'+digit))
			if dot {
				scale++
			}
			continue
		}
		if c == '.' {
			if dot {
				panic(bigNumberFormatException("Character array contains more than one decimal point."))
			}
			dot = true
			continue
		}
		if c == 'e' || c == 'E' {
			scale -= parseBigDecimalExponent(chars, i+1)
			break
		}
		if compact {
			message := ConcatJavaStrings(ConcatJavaStrings(bigNumberJavaString("Character "), NewJavaStringUTF16([]uint16{c})), bigNumberJavaString(" is neither a decimal digit number, decimal point, nor \"e\" notation exponential mark."))
			panic(NewJavaNumberFormatException(message))
		}
		panic(bigNumberFormatException("Character array is missing \"e\" notation exponential mark."))
	}
	if len(digits) == 0 {
		panic(bigNumberFormatException("No digits found."))
	}
	if scale < math.MinInt32 || scale > math.MaxInt32 {
		panic(bigNumberFormatException("Exponent overflow."))
	}
	var coefficient big.Int
	coefficient.SetString(string(digits), 10)
	if negative {
		coefficient.Neg(&coefficient)
	}
	return coefficient, int32(scale)
}
func parseBigDecimalExponent(chars []uint16, start int) int64 {
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
	for len(chars)-start > 10 && bigNumberDecimalDigit(chars[start]) == 0 {
		start++
	}
	if len(chars)-start > 10 {
		panic(bigNumberFormatException("Too many nonzero exponent digits."))
	}
	var exponent int64
	for _, c := range chars[start:] {
		digit := bigNumberDecimalDigit(c)
		if digit < 0 {
			panic(bigNumberFormatException("Not a digit."))
		}
		exponent = exponent*10 + int64(digit)
	}
	if negative {
		return -exponent
	}
	return exponent
}
