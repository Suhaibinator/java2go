package transpiler

// Java Throwable is a concrete class even though reference signatures use the
// runtime Throwable interface. Physical superclass storage and its selector
// must name the same concrete runtime type, resolved before stripping owners.
func builtinExceptionStorageTypeName(javaType string, ctx Ctx) (string, bool) {
	canonical, known := builtinThrowableReferenceName(javaType, ctx)
	if !known {
		return "", false
	}
	name := stripJavaQualifier(canonical)
	if _, concrete := builtinExceptionTypes[name]; !concrete {
		return "", false
	}
	if canonical == "java.lang.Throwable" {
		return "ThrowableObject", true
	}
	return name, true
}
