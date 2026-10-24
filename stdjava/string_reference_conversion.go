package stdjava

// javaReferenceStringer is the new Java-facing virtual toString result ABI.
// It deliberately differs from the native Go string-returning protocol until
// the compiler and all Java String boundaries migrate together.
type javaReferenceStringer interface {
	StringJava2goExecution(*Execution) *JavaString
}

// JavaStringValueOfExecution implements the reference-preserving Object
// conversion boundary for the new String core and migrated source protocols.
// JDK21 String.valueOf(Object) returns obj.toString directly, including a null
// result. Do not rebox, intern, or render that returned reference through fmt.
func JavaStringValueOfExecution(execution *Execution, value any) *JavaString {
	if javaReferenceIsNull(value) {
		return JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'})
	}
	if stringValue, ok := value.(*JavaString); ok {
		return stringValue
	}
	value = collectionObjectView(value)
	if _, throwable := value.(nominalThrowableText); throwable {
		return JavaThrowableToStringExecution(execution, value)
	}
	if registeredJavaSourceValue(value) {
		if rendered, found := callRegisteredSourceJavaString(execution, value); found {
			return rendered
		}
		if registeredSourceObjectToString(value) {
			return ObjectDefaultJavaStringExecution(execution, value)
		}
		// An unmarked source value has no migrated default adapter. A similarly
		// named ordinary method is never evidence of Java toString ownership.
		panic(NewUnsupportedOperationException("Java source String conversion requires a registered reference-returning toString adapter"))
	}
	if rendered, found := javaStringValueOfNativeExecution(execution, value); found {
		return rendered
	}
	if source, ok := value.(javaReferenceStringer); ok {
		return source.StringJava2goExecution(execution)
	}
	if builder, ok := value.(*StringBuilder); ok {
		return builder.ToJavaString()
	}
	// The old native StringJava2goExecution/String() path loses reference
	// identity. Other runtime/source classes must acquire a reviewed Java-facing
	// adapter in the coordinated ABI migration instead of silently using it.
	panic(NewUnsupportedOperationException("Java String conversion requires a reference-returning toString adapter"))
}
