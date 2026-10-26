package stdjava

// JavaIntegerParseInt implements Integer.parseInt(String[, radix]) directly on
// UTF16 units. The indexed CharSequence overload is a separate Java contract.
func JavaIntegerParseInt(text *JavaString, radix ...int32) int32 {
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
	negative, index, limit := false, 0, int32(-2147483647)
	if first := text.units[0]; first < '0' {
		if first == '-' {
			negative, limit = true, -2147483648
		} else if first != '+' {
			panic(javaIntegerParseInputException(text, base))
		}
		index++
		if index == len(text.units) {
			panic(javaIntegerParseInputException(text, base))
		}
	}
	// Negative accumulation includes MIN_VALUE without first overflowing a
	// positive int. Both multiplication and subtraction are checked beforehand.
	minimumBeforeMultiply := limit / base
	var result int32
	for ; index < len(text.units); index++ {
		digit := javaIntegerDigit21(text.units[index])
		if digit < 0 || digit >= base || result < minimumBeforeMultiply {
			panic(javaIntegerParseInputException(text, base))
		}
		result *= base
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

// Input is appended as original UTF16 units, never encoded as host text.
func javaIntegerParseInputException(text *JavaString, radix int32) NumberFormatException {
	units := []uint16{'F', 'o', 'r', ' ', 'i', 'n', 'p', 'u', 't', ' ', 's', 't', 'r', 'i', 'n', 'g', ':', ' ', '"'}
	units = append(units, text.units...)
	units = append(units, '"')
	if radix != 10 {
		units = append(units, ' ', 'u', 'n', 'd', 'e', 'r', ' ', 'r', 'a', 'd', 'i', 'x', ' ')
		units = append(units, JavaStringValueOfInt(radix).units...)
	}
	return NewJavaNumberFormatException(&JavaString{units: units})
}

// JDK21's Character.digit(char,36) domain is pinned by the exhaustive 65536
// unit oracle. Using explicit BMP blocks avoids drift with Go Unicode tables.
// Surrogates are absent: parseInt reads chars, not supplementary codepoints.
func javaIntegerDigit21(unit uint16) int32 {
	switch {
	case unit >= '0' && unit <= '9':
		return int32(unit - '0')
	case unit >= 'A' && unit <= 'Z':
		return int32(unit-'A') + 10
	case unit >= 'a' && unit <= 'z':
		return int32(unit-'a') + 10
	case unit >= 0xFF21 && unit <= 0xFF3A:
		return int32(unit-0xFF21) + 10
	case unit >= 0xFF41 && unit <= 0xFF5A:
		return int32(unit-0xFF41) + 10
	}
	for _, zero := range [...]uint16{
		0x0660, 0x06F0, 0x07C0, 0x0966, 0x09E6, 0x0A66, 0x0AE6,
		0x0B66, 0x0BE6, 0x0C66, 0x0CE6, 0x0D66, 0x0DE6, 0x0E50,
		0x0ED0, 0x0F20, 0x1040, 0x1090, 0x17E0, 0x1810, 0x1946,
		0x19D0, 0x1A80, 0x1A90, 0x1B50, 0x1BB0, 0x1C40, 0x1C50,
		0xA620, 0xA8D0, 0xA900, 0xA9D0, 0xA9F0, 0xAA50, 0xABF0, 0xFF10,
	} {
		if unit < zero {
			break
		}
		if unit <= zero+9 {
			return int32(unit - zero)
		}
	}
	return -1
}
