package stdjava

// NewJavaClassCastExceptionMessage implements () and (String), retaining the
// immutable Java message reference and leaving initCause available, even null.
func NewJavaClassCastExceptionMessage(message *JavaString) ClassCastException {
	return ClassCastException{newJavaThrowableBase("ClassCastException", message)}
}
