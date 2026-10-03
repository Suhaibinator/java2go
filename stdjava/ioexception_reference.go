package stdjava

// NewJavaIOExceptionMessage preserves the String/no-argument constructor's
// exact nullable detailMessage and its initially available cause slot.
func NewJavaIOExceptionMessage(message *JavaString) IOException {
	return IOException{newJavaThrowableBase("IOException", message)}
}

// NewJavaIOExceptionCauseExecution invokes the cause's virtual toString in the
// original caller execution. An abrupt callback propagates before allocation.
func NewJavaIOExceptionCauseExecution(execution *Execution, cause any) IOException {
	return IOException{newJavaThrowableCause(execution, "IOException", cause)}
}

func NewJavaIOExceptionMessageCause(message *JavaString, cause any) IOException {
	base := newJavaThrowableBase("IOException", message)
	if !javaReferenceIsNull(cause) {
		base.state.cause = cause
	}
	base.state.causeInitialized = true
	return IOException{base}
}
