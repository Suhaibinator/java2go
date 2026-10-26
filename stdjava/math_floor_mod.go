package stdjava

// MathFloorModInt is java.lang.Math.floorMod(int, int).
func MathFloorModInt(dividend, divisor int32) int32 {
	if divisor == 0 {
		panic(NewArithmeticException("/ by zero"))
	}
	remainder := dividend % divisor
	if (dividend^divisor) < 0 && remainder != 0 {
		remainder += divisor
	}
	return remainder
}

// MathFloorModLong is java.lang.Math.floorMod(long, long).
func MathFloorModLong(dividend, divisor int64) int64 {
	if divisor == 0 {
		panic(NewArithmeticException("/ by zero"))
	}
	remainder := dividend % divisor
	if (dividend^divisor) < 0 && remainder != 0 {
		remainder += divisor
	}
	return remainder
}

// MathFloorModLongInt retains the JDK's int result for its mixed-width overload.
// The remainder has the divisor's sign and magnitude below its int magnitude.
func MathFloorModLongInt(dividend int64, divisor int32) int32 {
	return int32(MathFloorModLong(dividend, int64(divisor)))
}
