package stdjava

// JavaLongParseLong implements Long.parseLong(String[, radix]) directly on
// UTF16 units. The indexed CharSequence overload is a separate Java contract.
func JavaLongParseLong(text *JavaString, radix ...int32) int64 {
	if text == nil {
		panic(NewJavaNumberFormatException(JavaStringLiteralUTF16([]uint16{
			'C', 'a', 'n', 'n', 'o', 't', ' ', 'p', 'a', 'r', 's', 'e', ' ', 'n', 'u', 'l', 'l', ' ', 's', 't', 'r', 'i', 'n', 'g',
		})))
	}
	base := int32(10)
	if len(radix) != 0 {
		base = radix[0]
	}
	if base < 2 || base > 36 {
		units := []uint16{'r', 'a', 'd', 'i', 'x', ' '}
		units = append(units, JavaStringValueOfInt(base).units...)
		if base < 2 {
			units = append(units, ' ', 'l', 'e', 's', 's', ' ', 't', 'h', 'a', 'n', ' ', 'C', 'h', 'a', 'r', 'a', 'c', 't', 'e', 'r', '.', 'M', 'I', 'N', '_', 'R', 'A', 'D', 'I', 'X')
		} else {
			units = append(units, ' ', 'g', 'r', 'e', 'a', 't', 'e', 'r', ' ', 't', 'h', 'a', 'n', ' ', 'C', 'h', 'a', 'r', 'a', 'c', 't', 'e', 'r', '.', 'M', 'A', 'X', '_', 'R', 'A', 'D', 'I', 'X')
		}
		panic(NewJavaNumberFormatException(&JavaString{units: units}))
	}
	if len(text.units) == 0 {
		panic(javaIntegerParseInputException(text, base))
	}
	negative, index, limit := false, 0, int64(-9223372036854775807)
	if first := text.units[0]; first < '0' {
		if first == '-' {
			negative, limit = true, -9223372036854775808
		} else if first != '+' {
			panic(javaIntegerParseInputException(text, base))
		}
		index++
		if index == len(text.units) {
			panic(javaIntegerParseInputException(text, base))
		}
	}
	// Negative accumulation includes MIN_VALUE without first overflowing a
	// positive long. Both multiplication and subtraction are checked beforehand.
	minimumBeforeMultiply := limit / int64(base)
	var result int64
	for ; index < len(text.units); index++ {
		digit := int64(javaIntegerDigit21(text.units[index]))
		if digit < 0 || digit >= int64(base) || result < minimumBeforeMultiply {
			panic(javaIntegerParseInputException(text, base))
		}
		result *= int64(base)
		if result < limit+digit {
			panic(javaIntegerParseInputException(text, base))
		}
		result -= digit
	}
	if negative {
		return result
	}
	return -result
}

