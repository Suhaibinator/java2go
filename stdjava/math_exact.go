package stdjava

// MathAddExact rejects overflow at the selected Java int or long width before
// the wrapped sum can be stored or consumed by subsequent expressions.
func MathAddExact[T ~int32 | ~int64](left, right T) T {
	result := left + right
	if (left^result)&(right^result) < 0 {
		message := "integer overflow"
		one := T(1)
		if int64(one<<31) > 0 {
			message = "long overflow"
		}
		panic(NewArithmeticException(message))
	}
	return result
}

// MathMultiplyExact implements Java's int/int, long/long and long/int checked
// multiplication after overload lowering has selected the promoted width.
func MathMultiplyExact[T ~int32 | ~int64](left, right T) T {
	result := left * right
	one := T(1)
	minimum := (-one) << 31
	message := "integer overflow"
	if int64(one<<31) > 0 {
		longMinimum := int64(-1) << 63
		minimum = T(longMinimum)
		message = "long overflow"
	}
	if (left == minimum && right == -one) || (right == minimum && left == -one) || (left != 0 && result/left != right) {
		panic(NewArithmeticException(message))
	}
	return result
}
