package stdjava

// ClassCast applies the runtime Class token's membership test before projecting
// the Java reference into its receiver-declared physical result representation.
// The receiver is checked here, after both caller expressions have run.
func ClassCast[T any](class *Class, value any) T {
	requested := class.TypeID()
	if nilJavaReference(value) {
		return ObjectView[T](value, requested)
	}
	actual, known := ObjectDynamicType(value)
	if isPrimitiveTypeID(requested) || requested == "void" ||
		((!known || !JavaTypeAssignable(actual, requested)) && !nativeJavaInterfaceAssignable(value, requested)) {
		panic(classCastException(actual, requested))
	}
	return ObjectView[T](value, requested)
}

// Cast is the erased Class API; generated typed invocations use ClassCast.
func (class *Class) Cast(value any) any { return ClassCast[any](class, value) }

func classCastException(actual, requested TypeID) ClassCastException {
	if actual == "" {
		// No Java class name can be invented for an unsupported opaque host value.
		return referenceCastException(actual, requested)
	}
	return NewClassCastException("Cannot cast " + javaClassBinaryName(actual) + " to " + javaClassBinaryName(requested))
}
