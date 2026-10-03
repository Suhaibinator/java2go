package stdjava

import (
	"math"
	"strconv"
)

// Primitive conversion entry points encode Java's statically chosen overload.
// In particular rune/int32 are the same Go type but char/int differ in Java.
func JavaStringValueOfBoolean(value bool) *JavaString {
	if value {
		return JavaStringLiteralUTF16([]uint16{'t', 'r', 'u', 'e'})
	}
	return JavaStringLiteralUTF16([]uint16{'f', 'a', 'l', 's', 'e'})
}

func JavaStringValueOfChar(value rune) *JavaString {
	return NewJavaStringUTF16([]uint16{uint16(value)})
}

func JavaStringValueOfInt(value int32) *JavaString {
	return javaStringFromNumericASCII(strconv.FormatInt(int64(value), 10), false)
}

func JavaStringValueOfLong(value int64) *JavaString {
	return javaStringFromNumericASCII(strconv.FormatInt(value, 10), false)
}

func JavaStringValueOfFloat(value float32) *JavaString {
	// JDK21 FloatToDecimal.toDecimalString returns literals for zero and
	// nonfinite values, and a freshly constructed String for finite nonzero.
	literal := value == 0 || math.IsNaN(float64(value)) || math.IsInf(float64(value), 0)
	return javaStringFromNumericASCII(FloatToString(value), literal)
}

func JavaStringValueOfDouble(value float64) *JavaString {
	// DoubleToDecimal uses the same literal policy as FloatToDecimal. Retain
	// the existing Java-format kernel's negative zero and precision behavior.
	literal := value == 0 || math.IsNaN(value) || math.IsInf(value, 0)
	return javaStringFromNumericASCII(DoubleToString(value), literal)
}

// Only trusted ASCII numeric-format kernels call this helper; arbitrary host
// text requires an explicit encoding policy at the actual Java boundary.
func javaStringFromNumericASCII(text string, literal bool) *JavaString {
	units := make([]uint16, len(text))
	for index := range text {
		units[index] = uint16(text[index])
	}
	if literal {
		return JavaStringLiteralUTF16(units)
	}
	return &JavaString{units: units}
}
