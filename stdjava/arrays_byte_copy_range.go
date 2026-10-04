package stdjava

import "strconv"

// ArraysByteCopyOfRange implements the canonical byte[] overload, retaining
// JDK21's full-range fast-path exception order and signed int length arithmetic.
func ArraysByteCopyOfRange(original *PrimitiveArray[int8], from, to int32) *PrimitiveArray[int8] {
	if from == 0 && to == PrimitiveArrayLength(original) {
		result := NewPrimitiveArray[int8](to, PrimitiveByteTypeID)
		copy(result.Elements, original.Elements)
		return result
	}
	if to < from {
		panic(NewIllegalArgumentException(strconv.FormatInt(int64(from), 10) + " > " + strconv.FormatInt(int64(to), 10)))
	}
	length := to - from
	if length < 0 {
		panic(NewNegativeArraySizeException(strconv.FormatInt(int64(length), 10)))
	}
	result := NewPrimitiveArray[int8](length, PrimitiveByteTypeID)
	originalLength := PrimitiveArrayLength(original)
	if from < 0 || from > originalLength {
		panic(NewArrayIndexOutOfBoundsException(""))
	}
	count := min(originalLength-from, length)
	copy(result.Elements, original.Elements[int(from):int(from+count)])
	return result
}
