package stdjava

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// StringValueOf implements the text conversion shared by Java's
// String.valueOf(Object) and the primitive print/concatenation operations.
// In particular, Java keeps a decimal point for whole floating-point values,
// spells infinities without Go's +/-Inf notation, and renders null as "null".
// fmt.Stringer is intentionally honored so generated enums can expose Java's
// Enum.toString default without leaking their Go struct representation.
func StringValueOf(value any) string {
	if value == nil {
		return "null"
	}

	switch value := value.(type) {
	case float32:
		return FloatToString(value)
	case float64:
		return DoubleToString(value)
	default:
		return fmt.Sprint(value)
	}
}

// FloatToString formats a float32 according to Java's Float.toString rules.
func FloatToString(value float32) string {
	return javaFloatingPointString(float64(value), 32)
}

// DoubleToString formats a float64 according to Java's Double.toString rules.
func DoubleToString(value float64) string {
	return javaFloatingPointString(value, 64)
}

func javaFloatingPointString(value float64, bitSize int) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "Infinity"
	case math.IsInf(value, -1):
		return "-Infinity"
	}

	abs := math.Abs(value)
	if abs == 0 || (abs >= 1e-3 && abs < 1e7) {
		formatted := strconv.FormatFloat(value, 'f', -1, bitSize)
		if !strings.ContainsRune(formatted, '.') {
			formatted += ".0"
		}
		return formatted
	}

	formatted := strconv.FormatFloat(value, 'e', -1, bitSize)
	mantissa, exponentText, found := strings.Cut(formatted, "e")
	if !found {
		return formatted
	}
	if !strings.ContainsRune(mantissa, '.') {
		// Java's scientific form requires a fractional digit. Appending zero to
		// Go's one-digit shortest form can choose a decimal farther from the
		// binary value (Float.MIN_VALUE would become 1.0E-45). Re-round with one
		// fractional digit so Java's closest qualifying form is retained.
		formatted = strconv.FormatFloat(value, 'e', 1, bitSize)
		mantissa, exponentText, _ = strings.Cut(formatted, "e")
	}
	exponent, err := strconv.Atoi(exponentText)
	if err != nil {
		return mantissa + "E" + exponentText
	}
	return mantissa + "E" + strconv.Itoa(exponent)
}
