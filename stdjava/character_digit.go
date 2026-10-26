package stdjava

import "unicode"

// CharDigit implements Character.digit for both char and Unicode code points.
// Besides decimal digits, Java accepts ASCII and fullwidth Latin letters.
func CharDigit(character rune, radix int32) int32 {
	if radix < 2 || radix > 36 {
		return -1
	}
	var value int32 = -1
	switch {
	case character >= '0' && character <= '9':
		value = character - '0'
	case character >= 'a' && character <= 'z':
		value = character - 'a' + 10
	case character >= 'A' && character <= 'Z':
		value = character - 'A' + 10
	case character >= 'ａ' && character <= 'ｚ':
		value = character - 'ａ' + 10
	case character >= 'Ａ' && character <= 'Ｚ':
		value = character - 'Ａ' + 10
	case character >= 0 && character <= 0xffff:
		for _, span := range unicode.Digit.R16 {
			c := uint16(character)
			if c >= span.Lo && c <= span.Hi && (c-span.Lo)%span.Stride == 0 {
				value = int32((c-span.Lo)/span.Stride) % 10
				break
			}
		}
	case character > 0xffff && character <= unicode.MaxRune:
		for _, span := range unicode.Digit.R32 {
			c := uint32(character)
			if c >= span.Lo && c <= span.Hi && (c-span.Lo)%span.Stride == 0 {
				value = int32((c-span.Lo)/span.Stride) % 10
				break
			}
		}
	}
	if value >= radix {
		return -1
	}
	return value
}

// CharForDigit returns the lowercase ASCII digit for a radix, or the NUL char
// when the digit or radix is outside Character.forDigit's accepted range.
func CharForDigit(digit, radix int32) rune {
	if radix < 2 || radix > 36 || digit < 0 || digit >= radix {
		return 0
	}
	if digit < 10 {
		return '0' + digit
	}
	return 'a' + digit - 10
}
