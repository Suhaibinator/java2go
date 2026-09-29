package stdjava

// NewJavaIllegalStateExceptionMessage implements the String overload. A nil
// message retains Java null and leaves the cause available for later initCause.
// The no-argument constructor has the same initial detail-message/cause state.
func NewJavaIllegalStateExceptionMessage(message *JavaString) IllegalStateException {
	return IllegalStateException{newJavaThrowableBase("IllegalStateException", message)}
}

// NewJavaExceptionMessage preserves the original immutable message reference.
// Keep this entry separate from the cause overload, including for typed null.
func NewJavaExceptionMessage(message *JavaString) Exception {
	return Exception{newJavaThrowableBase("Exception", message)}
}
