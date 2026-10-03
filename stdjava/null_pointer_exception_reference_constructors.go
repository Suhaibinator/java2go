package stdjava

// NewJavaNullPointerExceptionMessage implements the Java String and no-argument
// constructors. A nil message retains Java null and leaves initCause available;
// a non-nil message is retained as the original immutable Java String reference.
func NewJavaNullPointerExceptionMessage(message *JavaString) NullPointerException {
	return NullPointerException{newJavaThrowableBase("NullPointerException", message)}
}
