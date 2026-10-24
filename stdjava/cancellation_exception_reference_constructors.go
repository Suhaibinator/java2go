package stdjava

// NewJavaCancellationExceptionMessage preserves the JDK no-argument/String
// declarations: nullable exact String identity and an available initCause slot.
func NewJavaCancellationExceptionMessage(message *JavaString) CancellationException {
	return CancellationException{newJavaThrowableBase("CancellationException", message)}
}
