package stdjava

// NewJavaIllegalArgumentExceptionMessage implements the String and no-argument
// overloads without consuming the cause slot or copying the message reference.
func NewJavaIllegalArgumentExceptionMessage(message *JavaString) IllegalArgumentException {
	return IllegalArgumentException{newJavaThrowableBase("IllegalArgumentException", message)}
}

// NewJavaIllegalArgumentExceptionCauseExecution implements the Throwable
// overload, including virtual toString and a typed-null initialized cause.
func NewJavaIllegalArgumentExceptionCauseExecution(execution *Execution, cause any) IllegalArgumentException {
	return IllegalArgumentException{newJavaThrowableCause(execution, "IllegalArgumentException", cause)}
}

// NewJavaIllegalArgumentExceptionMessageCause implements (String, Throwable).
// An explicit cause, including null, closes the cause slot without rendering it.
func NewJavaIllegalArgumentExceptionMessageCause(message *JavaString, cause any) IllegalArgumentException {
	base := newJavaThrowableBase("IllegalArgumentException", message)
	if !javaReferenceIsNull(cause) {
		base.state.cause = cause
	}
	base.state.causeInitialized = true
	return IllegalArgumentException{base}
}
