package stdjava

// NewJavaIndexOutOfBoundsExceptionMessage preserves the String overload's
// original message reference, including null, with initCause still available.
func NewJavaIndexOutOfBoundsExceptionMessage(message *JavaString) IndexOutOfBoundsException {
	return IndexOutOfBoundsException{newJavaThrowableBase("IndexOutOfBoundsException", message)}
}

// NewJavaParseExceptionMessage keeps both Java constructor parameters without
// converting the immutable UTF16 message to native text or initializing cause.
func NewJavaParseExceptionMessage(message *JavaString, offset int32) ParseException {
	return ParseException{newJavaThrowableBase("ParseException", message), offset}
}
