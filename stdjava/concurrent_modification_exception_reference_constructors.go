package stdjava

// NewJavaConcurrentModificationExceptionMessage retains the nullable String
// detail reference and leaves the cause slot available, including for noarg.
func NewJavaConcurrentModificationExceptionMessage(message *JavaString) ConcurrentModificationException {
	return ConcurrentModificationException{newJavaThrowableBase("ConcurrentModificationException", message)}
}

// NewJavaConcurrentModificationExceptionCauseExecution renders a nonnull cause
// once in the original caller execution before constructing the exception.
func NewJavaConcurrentModificationExceptionCauseExecution(execution *Execution, cause any) ConcurrentModificationException {
	return ConcurrentModificationException{newJavaThrowableCause(execution, "ConcurrentModificationException", cause)}
}

func NewJavaConcurrentModificationExceptionMessageCause(message *JavaString, cause any) ConcurrentModificationException {
	base := newJavaThrowableBase("ConcurrentModificationException", message)
	if !javaReferenceIsNull(cause) {
		base.state.cause = cause
	}
	base.state.causeInitialized = true
	return ConcurrentModificationException{base}
}
