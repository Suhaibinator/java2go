package stdjava

import (
	"math"
	"strconv"
	"strings"
	"unicode"
)

// Parsing methods return primitives; valueOf methods in boxed_factories.go
// return nullable wrapper objects. Invalid numeric strings throw Java's
// NumberFormatException rather than a Go strconv error.
func ParseByte(s string, radix ...int32) int8   { return int8(parseInteger(s, 8, radix)) }
func ParseShort(s string, radix ...int32) int16 { return int16(parseInteger(s, 16, radix)) }
func ParseInt(s string, radix ...int32) int32   { return int32(parseInteger(s, 32, radix)) }
func ParseLong(s string, radix ...int32) int64  { return parseInteger(s, 64, radix) }

func parseInteger(s string, bits int, radix []int32) int64 {
	base := int32(10)
	if len(radix) != 0 {
		base = radix[0]
	}
	if len(radix) > 1 || base < 2 || base > 36 || StringIsNull(s) {
		panic(NewNumberFormatException("invalid radix or null string"))
	}
	// Integer.parseInt uses Character.digit on UTF-16 code units. Normalize
	// supported BMP decimal digits and fullwidth letters before strconv parsing.
	normalized := strings.Map(func(character rune) rune {
		if character >= '\uff10' && character <= '\uff19' {
			return '0' + character - '\uff10'
		}
		if character >= '\uff21' && character <= '\uff3a' {
			return 'A' + character - '\uff21'
		}
		if character >= '\uff41' && character <= '\uff5a' {
			return 'a' + character - '\uff41'
		}
		if character > 127 && character <= 0xffff {
			for _, span := range unicode.Digit.R16 {
				if uint16(character) >= span.Lo && uint16(character) <= span.Hi && (uint16(character)-span.Lo)%span.Stride == 0 {
					return '0' + rune((uint16(character)-span.Lo)/span.Stride)%10
				}
			}
		}
		return character
	}, s)
	value, err := strconv.ParseInt(normalized, int(base), bits)
	if err != nil {
		panic(NewNumberFormatException("For input string: " + strconv.Quote(s)))
	}
	return value
}

func ParseFloat(s string) float32  { return float32(parseFloatingPoint(s, 32)) }
func ParseDouble(s string) float64 { return parseFloatingPoint(s, 64) }

func parseFloatingPoint(s string, bits int) float64 {
	if StringIsNull(s) {
		panic(NewNullPointerException("floating-point string is null"))
	}
	text := strings.TrimFunc(s, func(character rune) bool { return character <= ' ' })
	switch text {
	case "NaN", "+NaN", "-NaN":
		return math.NaN()
	case "Infinity", "+Infinity":
		return math.Inf(1)
	case "-Infinity":
		return math.Inf(-1)
	}
	if len(text) > 0 {
		suffix := text[len(text)-1]
		if suffix == 'f' || suffix == 'F' || suffix == 'd' || suffix == 'D' {
			text = text[:len(text)-1]
		}
	}
	// Go accepts Inf, case-insensitive infinity, and underscores; Java's
	// runtime parser accepts none of those forms. A numeric first digit or
	// decimal point also excludes suffixed NaN and Infinity.
	numeric := text
	if len(numeric) > 0 && (numeric[0] == '+' || numeric[0] == '-') {
		numeric = numeric[1:]
	}
	if len(numeric) == 0 || (numeric[0] != '.' && (numeric[0] < '0' || numeric[0] > '9')) || strings.ContainsRune(text, '_') {
		panic(NewNumberFormatException("For input string: " + strconv.Quote(s)))
	}
	value, err := strconv.ParseFloat(text, bits)
	if err != nil {
		if numberError, ok := err.(*strconv.NumError); !ok || numberError.Err != strconv.ErrRange {
			panic(NewNumberFormatException("For input string: " + strconv.Quote(s)))
		}
	}
	return value
}

// ParseBoolean returns false for null and every string other than true,
// ignoring case, as required by Boolean.parseBoolean.
func ParseBoolean(s string) bool {
	return len(s) == 4 &&
		(s[0] == 't' || s[0] == 'T') &&
		(s[1] == 'r' || s[1] == 'R') &&
		(s[2] == 'u' || s[2] == 'U') &&
		(s[3] == 'e' || s[3] == 'E')
}
