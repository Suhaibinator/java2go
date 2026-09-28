package stdjava

// MathAddExact rejects overflow at the selected Java int or long width before
// the wrapped sum can be stored or consumed by subsequent expressions.
func MathAddExact[T ~int32 | ~int64](left, right T) T {
	result := left + right
	if (left^result)&(right^result) < 0 {
		message := "integer overflow"
		one := T(1)
		if one<<32 != 0 {
			message = "long overflow"
		}
		panic(NewArithmeticException(message))
	}
	return result
}
