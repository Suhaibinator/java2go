package stdjava

// ReferenceCheckCast implements a Java reference cast whose written target
// carries a proved nominal descriptor. The physical Go carrier is not evidence
// of Java class membership: distinct Java classes may share it.
func ReferenceCheckCast[T any](value any, requested TypeID) T {
	if !nilJavaReference(value) {
		actual, known := ObjectDynamicType(value)
		// Preserve the runtime's explicit private interface membership proofs.
		// An otherwise opaque value must not use ObjectView's broad Object escape
		// to acquire an identity that its allocation never supplied.
		if !known && !nativeJavaInterfaceAssignable(value, requested) {
			panic(referenceCastException(actual, requested))
		}
	}
	return ObjectView[T](value, requested)
}
