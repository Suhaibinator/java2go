package stdjava

// MathToIntExact performs Java's checked long-to-int narrowing conversion.
func MathToIntExact(value int64) int32 {
	result := int32(value)
	if int64(result) != value {
		panic(NewArithmeticException("integer overflow"))
	}
	return result
}
