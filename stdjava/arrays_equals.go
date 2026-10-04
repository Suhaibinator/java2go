package stdjava

import "math"

// ArraysEqualsExecution is Java's shallow two-array equality operation. Source
// equals methods run in the caller's logical execution; nested arrays retain
// identity semantics instead of recursively comparing their contents.
func ArraysEqualsExecution(execution *Execution, left, right any) bool {
	if JavaReferenceEqual(left, right) {
		return true
	}
	if javaReferenceIsNull(left) || javaReferenceIsNull(right) {
		return false
	}
	switch array := left.(type) {
	case *PrimitiveArray[bool]:
		return arraysEqualPrimitive(array, right, nil)
	case *PrimitiveArray[int8]:
		return arraysEqualPrimitive(array, right, nil)
	case *PrimitiveArray[int16]:
		return arraysEqualPrimitive(array, right, nil)
	case *PrimitiveArray[int32]:
		return arraysEqualPrimitive(array, right, nil)
	case *PrimitiveArray[int64]:
		return arraysEqualPrimitive(array, right, nil)
	case *PrimitiveArray[float32]:
		return arraysEqualPrimitive(array, right, func(a, b float32) bool {
			return math.Float32bits(a) == math.Float32bits(b) || (math.IsNaN(float64(a)) && math.IsNaN(float64(b)))
		})
	case *PrimitiveArray[float64]:
		return arraysEqualPrimitive(array, right, func(a, b float64) bool {
			return math.Float64bits(a) == math.Float64bits(b) || (math.IsNaN(a) && math.IsNaN(b))
		})
	case *ReferenceArray:
		other, ok := right.(*ReferenceArray)
		if !ok || len(array.elements) != len(other.elements) {
			return false
		}
		for index, element := range array.elements {
			if !ObjectsEqual(element, other.elements[index], execution) {
				return false
			}
		}
		return true
	default:
		panic(NewUnsupportedOperationException("Arrays.equals requires Java arrays"))
	}
}

func arraysEqualPrimitive[T comparable](left *PrimitiveArray[T], right any, equal func(T, T) bool) bool {
	other, ok := right.(*PrimitiveArray[T])
	if !ok || left.componentType != other.componentType || len(left.Elements) != len(other.Elements) {
		return false
	}
	for index, value := range left.Elements {
		if equal != nil {
			if !equal(value, other.Elements[index]) {
				return false
			}
		} else if value != other.Elements[index] {
			return false
		}
	}
	return true
}
