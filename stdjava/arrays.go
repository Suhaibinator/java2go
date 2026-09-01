package stdjava

// javaArrayLength contains the integral Go types a transpiled Java array
// dimension can have before or after unary numeric promotion.
type javaArrayLength interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint8 | ~uint16
}

// NewArray allocates the backing slice for a Java array. Java gives every array
// allocation its own object identity, including arrays of length zero. A plain
// make([]T, 0) is allowed to reuse Go's global zero base, so retain one element
// of capacity while exposing the requested Java length. Positive-sized arrays
// keep the ordinary len==cap shape and pay no additional storage cost.
func NewArray[T any, I javaArrayLength](length I) []T {
	if length < 0 {
		panic(NewNegativeArraySizeException(""))
	}
	size := int(length)
	capacity := size
	if capacity == 0 {
		capacity = 1
	}
	array := make([]T, size, capacity)
	// Java String is a nullable reference even though generated code retains a
	// concrete Go string ABI. Populate every visible element with the null String
	// sentinel so new String[n] differs from n empty strings. Multidimensional
	// lowering allocates leaf rows through NewArray[string], inheriting the same
	// rule without special casing generated loop structure.
	var zero T
	if _, stringElements := any(zero).(string); stringElements {
		nullString := any(NullString()).(T)
		for index := range array {
			array[index] = nullString
		}
	}
	return array
}

// ArrayLiteral preserves Java array identity for initializer expressions.
// Non-empty variadic arguments already have unique backing storage. The empty
// form needs the same retained single-element capacity as NewArray.
func ArrayLiteral[T any](elements ...T) []T {
	if len(elements) == 0 {
		return make([]T, 0, 1)
	}
	return elements
}

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
