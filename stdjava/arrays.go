package stdjava

// ArraySet implements the evaluation/check/store boundary of a Java simple
// array assignment. Generated code evaluates array, index, and value as call
// arguments (in that order) before this function performs the null check,
// bounds check, and store. Keeping the checks here also distinguishes a null
// Java array from an empty Go slice, which would otherwise both produce Go's
// native index-out-of-range panic.
func ArraySet[T any, I ~int | ~int8 | ~int16 | ~int32](array []T, index I, value T) T {
	if array == nil {
		panic(NewNullPointerException("array assignment on null"))
	}
	index64 := int64(index)
	if index64 < 0 || index64 >= int64(len(array)) {
		panic(NewArrayIndexOutOfBoundsException("array index out of bounds"))
	}
	array[index] = value
	return value
}
