package stdjava

// ObjectsRequireNonNull returns the same Java reference, including its static
// generic representation, or throws when that reference is null.
func ObjectsRequireNonNull[T any](value T) T {
	if javaReferenceIsNull(value) {
		panic(NewNullPointerException(NullString()))
	}
	return value
}

func ObjectsRequireNonNullMessage[T any](value T, message string) T {
	if javaReferenceIsNull(value) {
		panic(NewNullPointerException(message))
	}
	return value
}

func ObjectsRequireNonNullSupplier[T any](execution *Execution, value T, supplier Supplier[string]) T {
	if javaReferenceIsNull(value) {
		message := NullString()
		if !javaReferenceIsNull(supplier) {
			message = CallSupplierExecution[string](execution, supplier)
		}
		panic(NewNullPointerException(message))
	}
	return value
}
