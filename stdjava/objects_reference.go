package stdjava

// ObjectsRequireNonNullReference preserves the generated Java reference's exact
// static representation and throws a canonical null-message exception for null.
func ObjectsRequireNonNullReference[T any](value T) T {
	if javaReferenceIsNull(value) {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	return value
}

// ObjectsRequireNonNullMessageReference retains the original immutable Java
// String reference, including null, empty content, and every UTF16 code unit.
func ObjectsRequireNonNullMessageReference[T any](value T, message *JavaString) T {
	if javaReferenceIsNull(value) {
		panic(NewJavaNullPointerExceptionMessage(message))
	}
	return value
}

// ObjectsRequireNonNullSupplierReference evaluates get only for a null value.
// Callback dispatch keeps the caller's execution and propagates abrupt completion
// unchanged; a null supplier or null result produces a null exception message.
func ObjectsRequireNonNullSupplierReference[T any](execution *Execution, value T, supplier Supplier[*JavaString]) T {
	if javaReferenceIsNull(value) {
		var message *JavaString
		if !javaReferenceIsNull(supplier) {
			message = CallSupplierExecution[*JavaString](execution, supplier)
		}
		panic(NewJavaNullPointerExceptionMessage(message))
	}
	return value
}
