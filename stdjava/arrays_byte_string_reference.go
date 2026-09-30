package stdjava

// JavaByteArrayToString returns the canonical result of Arrays.toString(byte[]).
// The existing byte formatting kernel produces trusted numeric ASCII. JDK null
// and empty arrays return literals; nonempty arrays return a fresh String.
func JavaByteArrayToString(array *PrimitiveArray[int8]) *JavaString {
	literal := array == nil || len(array.Elements) == 0
	return javaStringFromNumericASCII(ArrayToString(array), literal)
}
