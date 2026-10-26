package stdjava

// StringIndexOutOfBoundsException preserves the JDK subtype independently of
// Go bounds panics, which otherwise normalize to an array exception.
type StringIndexOutOfBoundsException struct{ ThrowableBase }

func NewStringIndexOutOfBoundsException(message string) StringIndexOutOfBoundsException {
	return StringIndexOutOfBoundsException{newThrowableBase("StringIndexOutOfBoundsException", message)}
}
