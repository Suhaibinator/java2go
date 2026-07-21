package stdjava

import (
	"math"

	"golang.org/x/exp/constraints"
)

// Ternary represents Java's ternary operator (condition ? result1 : result2)
func Ternary[T any](condition bool, result1, result2 T) T {
	if condition {
		return result1
	}
	return result2
}

// UnsignedRightShift is an implementation of Java's unsigned right shift
// operation where a number is shifted over the number of times specified, but
// the topmost bits are always filled in with zeroes
func UnsignedRightShift[V, A constraints.Integer](value V, amount A) V {
	// Java applies >>> to either a 32-bit int or a 64-bit long. Generated Go
	// uses int32/int64 for those types; untyped/direct int calls retain the
	// historical Java-int behavior.
	switch any(value).(type) {
	case int64, uint64:
		return V(uint64(value) >> uint64(amount))
	default:
		return V(uint32(value) >> uint32(amount))
	}
}

// UnsignedRightShiftAssignment represents a right-shift assignment (`>>>=`)
// where a value is assigned the result of an unsigned right shift
func UnsignedRightShiftAssignment[V, A constraints.Integer](assignTo *V, amount A) {
	*assignTo = UnsignedRightShift(*assignTo, amount)
}

// number covers the Java primitive numeric types that support the ++ and --
// operators.
type number interface {
	constraints.Integer | constraints.Float
}

// PostIncrement implements Java's post-increment (`x++`) in expression position:
// it increments the pointed-to value and returns the value from before the
// increment.
func PostIncrement[T number](p *T) T {
	old := *p
	*p++
	return old
}

// PreIncrement implements Java's pre-increment (`++x`): it increments the
// pointed-to value and returns the new value.
func PreIncrement[T number](p *T) T {
	*p++
	return *p
}

// PostDecrement implements Java's post-decrement (`x--`): it decrements the
// pointed-to value and returns the value from before the decrement.
func PostDecrement[T number](p *T) T {
	old := *p
	*p--
	return old
}

// PreDecrement implements Java's pre-decrement (`--x`): it decrements the
// pointed-to value and returns the new value.
func PreDecrement[T number](p *T) T {
	*p--
	return *p
}

// HashCode is an implementation of Java's String `hashCode` method
func HashCode(s string) int {
	var total int
	n := len(s)
	for ind, char := range s {
		total += int(char) * int(math.Pow(float64(31), float64(n-(ind+1))))
	}
	return total
}

// MultiDimensionArray constructs an array with two dimensions
func MultiDimensionArray[T any](val []T, dims ...int) [][]T {
	arr := make([][]T, dims[0])
	for ind := range arr {
		arr[ind] = make([]T, dims[1])
	}
	return arr
}

// MultiDimensionArray3 constructs an array with three dimensions
func MultiDimensionArray3[T any](val [][]T, dims ...int) [][][]T {
	arr := make([][][]T, dims[0])
	for ind := range arr {
		arr[ind] = MultiDimensionArray([]T{}, dims[1:]...)
	}
	return arr
}
