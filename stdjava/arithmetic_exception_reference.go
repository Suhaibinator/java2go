package stdjava

// NewJavaArithmeticExceptionMessage implements () and (String), retaining the
// immutable Java message reference and leaving initCause available, even null.
func NewJavaArithmeticExceptionMessage(message *JavaString) ArithmeticException {
	return ArithmeticException{newJavaThrowableBase("ArithmeticException", message)}
}
