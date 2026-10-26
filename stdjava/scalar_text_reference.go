package stdjava

import (
	"math"
	"strconv"
	"strings"
)

// JavaByteParseByte and JavaShortParseShort use Integer's UTF16 parser before
// checking their narrower range, matching both exception order and diagnostics.
func JavaByteParseByte(text *JavaString, radix ...int32) int8 {
	value := JavaIntegerParseInt(text, radix...)
	if value < -128 || value > 127 {
		panic(javaNarrowIntegerRangeException(text, radix))
	}
	return int8(value)
}

func JavaShortParseShort(text *JavaString, radix ...int32) int16 {
	value := JavaIntegerParseInt(text, radix...)
	if value < -32768 || value > 32767 {
		panic(javaNarrowIntegerRangeException(text, radix))
	}
	return int16(value)
}

func javaNarrowIntegerRangeException(text *JavaString, radix []int32) NumberFormatException {
	base := int32(10)
	if len(radix) != 0 {
		base = radix[0]
	}
	units := []uint16{'V', 'a', 'l', 'u', 'e', ' ', 'o', 'u', 't', ' ', 'o', 'f', ' ', 'r', 'a', 'n', 'g', 'e', '.', ' ', 'V', 'a', 'l', 'u', 'e', ':', '"'}
	units = append(units, text.units...)
	units = append(units, '"', ' ', 'R', 'a', 'd', 'i', 'x', ':')
	units = append(units, JavaStringValueOfInt(base).units...)
	return NewJavaNumberFormatException(NewJavaStringUTF16(units))
}

func JavaBooleanParseBoolean(text *JavaString) bool {
	if text == nil || len(text.units) != 4 {
		return false
	}
	for index, expected := range []uint16{'t', 'r', 'u', 'e'} {
		unit := text.units[index]
		if unit != expected && unit != expected-('a'-'A') {
			return false
		}
	}
	return true
}

func JavaFloatParseFloat(text *JavaString) float32 {
	return float32(javaParseFloatingPoint(text, 32))
}

func JavaDoubleParseDouble(text *JavaString) float64 {
	return javaParseFloatingPoint(text, 64)
}

func javaParseFloatingPoint(text *JavaString, bits int) float64 {
	if text == nil {
		// FloatingDecimal's JDK21 boundary invokes String.trim before parsing.
		panic(NewJavaNullPointerExceptionMessage(JavaStringFromHostUTF8(`Cannot invoke "String.trim()" because "in" is null`)))
	}
	start, end := 0, len(text.units)
	for start < end && text.units[start] <= ' ' {
		start++
	}
	for end > start && text.units[end-1] <= ' ' {
		end--
	}
	units := text.units[start:end]
	if len(units) == 0 {
		panic(NewJavaNumberFormatException(JavaStringLiteralUTF16([]uint16{'e', 'm', 'p', 't', 'y', ' ', 'S', 't', 'r', 'i', 'n', 'g'})))
	}
	// Java's floating grammar is ASCII. Validate units before constructing the
	// numeric host text; rejected input stays in its original trimmed UTF16 form.
	bytes := make([]byte, len(units))
	for index, unit := range units {
		if unit > 127 {
			panic(javaFloatingPointInputException(units))
		}
		bytes[index] = byte(unit)
	}
	numeric := string(bytes)
	switch numeric {
	case "NaN", "+NaN", "-NaN":
		// JDK Float/Double parsers use the canonical quiet NaN payload.
		return math.Float64frombits(0x7ff8000000000000)
	case "Infinity", "+Infinity":
		return math.Inf(1)
	case "-Infinity":
		return math.Inf(-1)
	}
	if suffix := numeric[len(numeric)-1]; suffix == 'f' || suffix == 'F' || suffix == 'd' || suffix == 'D' {
		numeric = numeric[:len(numeric)-1]
	}
	unsigned := numeric
	if len(unsigned) != 0 && (unsigned[0] == '+' || unsigned[0] == '-') {
		unsigned = unsigned[1:]
	}
	// Exclude Go's special values and underscore syntax. ParseFloat then checks
	// the remaining decimal/hex grammar and rounds at the requested IEEE width.
	if len(unsigned) == 0 || unsigned[0] != '.' && (unsigned[0] < '0' || unsigned[0] > '9') || strings.ContainsRune(numeric, '_') {
		panic(javaFloatingPointInputException(units))
	}
	value, err := strconv.ParseFloat(numeric, bits)
	if err != nil {
		if numberError, ok := err.(*strconv.NumError); !ok || numberError.Err != strconv.ErrRange {
			panic(javaFloatingPointInputException(units))
		}
	}
	return value
}

func javaFloatingPointInputException(input []uint16) NumberFormatException {
	units := []uint16{'F', 'o', 'r', ' ', 'i', 'n', 'p', 'u', 't', ' ', 's', 't', 'r', 'i', 'n', 'g', ':', ' ', '"'}
	units = append(units, input...)
	units = append(units, '"')
	return NewJavaNumberFormatException(NewJavaStringUTF16(units))
}
