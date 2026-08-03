package stdjava

import "reflect"

// ReferenceRequireNonNull returns value without changing its exact Go type and
// throws Java's NullPointerException when value represents a null reference.
// Keeping T in both the parameter and result is important at generated generic
// boundaries: converting through any would erase the concrete instantiation
// that Go needs for type inference.
func ReferenceRequireNonNull[T any](value T) T {
	if javaReferenceIsNull(value) {
		panic(NewNullPointerException("null reference"))
	}
	return value
}

// ReferenceTypeHint widens value to an explicitly selected generated Java
// reference view while preserving Java null. Generated calls use the result
// type to guide Go's generic inference when Java erased a member-class type
// argument that Go must still instantiate.
func ReferenceTypeHint[T any](value T) T {
	if !javaReferenceIsNull(value) {
		return value
	}
	// A generated String slot uses NullString's sentinel because Go's concrete
	// string type has no nil value. Other reference views use their zero value.
	target := reflect.TypeOf((*T)(nil)).Elem()
	if target.Kind() == reflect.String {
		return value
	}
	var zero T
	return zero
}

// JavaReferenceEqual compares two values using the identity available in the
// generated Java-reference representations. It deliberately avoids Go's raw
// interface == operator: that operator treats a typed nil pointer as non-nil
// and panics when an interface contains a map, slice, function, or another
// non-comparable dynamic value.
//
// Generated source objects and arrays are pointer-backed, so pointer equality
// is their Java identity. Runtime maps/slices/channels use their stable backing
// pointer when they appear behind an erased interface. Function values do not
// expose closure-instance identity in Go; two non-nil functions are therefore
// conservatively distinct instead of being collapsed by their shared code
// pointer. String and boxed primitive values retain the value-backed identity
// approximation already used by the generated ABI.
func JavaReferenceEqual(left, right any) bool {
	leftNull := javaReferenceIsNull(left)
	rightNull := javaReferenceIsNull(right)
	if leftNull || rightNull {
		return leftNull && rightNull
	}

	leftValue := reflect.ValueOf(left)
	rightValue := reflect.ValueOf(right)
	if leftValue.Type() != rightValue.Type() {
		return false
	}

	switch leftValue.Kind() {
	case reflect.Pointer, reflect.UnsafePointer, reflect.Map, reflect.Chan:
		return leftValue.Pointer() == rightValue.Pointer()
	case reflect.Slice:
		leftPointer := leftValue.Pointer()
		rightPointer := rightValue.Pointer()
		// Go permits distinct non-nil empty slices to expose a zero backing
		// pointer. Treating both as identical would collapse distinct Java refs.
		return leftPointer != 0 && leftPointer == rightPointer
	case reflect.Func:
		// reflect.Value.Pointer reports the entry code address, not closure
		// identity; distinct closures commonly share it.
		return false
	}

	if leftValue.Comparable() {
		return leftValue.Interface() == rightValue.Interface()
	}
	return false
}

func javaReferenceIsNull(value any) bool {
	if StringIsNull(value) {
		return true
	}
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
