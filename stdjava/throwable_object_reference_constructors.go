package stdjava

// NewJavaThrowableMessage preserves a Java String message and leaves initCause
// available. The no-argument overload supplies a nil message to this entry.
func NewJavaThrowableMessage(message *JavaString) ThrowableObject {
	return ThrowableObject{newJavaThrowableBase("Throwable", message)}
}

// NewJavaThrowableCauseExecution evaluates the cause's virtual toString with
// the caller's execution and initializes the cause slot even for typed null.
func NewJavaThrowableCauseExecution(execution *Execution, cause any) ThrowableObject {
	return ThrowableObject{newJavaThrowableCause(execution, "Throwable", cause)}
}

// NewJavaThrowableMessageCause preserves the message and explicit cause. The
// cause slot is initialized even when the supplied cause is Java null.
func NewJavaThrowableMessageCause(message *JavaString, cause any) ThrowableObject {
	base := newJavaThrowableBase("Throwable", message)
	if !javaReferenceIsNull(cause) {
		base.state.cause = cause
	}
	base.state.causeInitialized = true
	return ThrowableObject{base}
}
