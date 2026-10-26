package stdjava

// JavaIntegerToHexString retains the native unsigned 32-bit formatting kernel
// and returns the canonical Java String allocation at the JDK method boundary.
func JavaIntegerToHexString(value int32) *JavaString {
	return javaStringFromNumericASCII(IntegerToHexString(value), false)
}
