package stdjava

import "unicode/utf16"

// Formatter failures share a representation but retain their distinct Java
// classes, canonical messages, and IllegalArgumentException ancestry.
type formatterException struct{ ThrowableBase }

func (failure formatterException) JavaDynamicTypeID() TypeID {
	return TypeID("java.util." + failure.ThrowableTypeName())
}

func javaFormatFailure(name, message string) {
	panic(formatterException{newJavaThrowableBase(name, NewJavaStringUTF16(utf16.Encode([]rune(message))))})
}

func javaFormatUnknownConversion(units []uint16) {
	message := append([]uint16{'C', 'o', 'n', 'v', 'e', 'r', 's', 'i', 'o', 'n', ' ', '=', ' ', '\''}, units...)
	message = append(message, '\'')
	panic(formatterException{newJavaThrowableBase("UnknownFormatConversionException", NewJavaStringUTF16(message))})
}

func init() {
	RegisterException("IllegalFormatException", "IllegalArgumentException")
	RegisterJavaType("java.util.IllegalFormatException", BuiltinThrowableTypeID("IllegalArgumentException"))
	for _, name := range []string{"UnknownFormatConversionException", "MissingFormatArgumentException", "DuplicateFormatFlagsException", "MissingFormatWidthException", "FormatFlagsConversionMismatchException", "IllegalFormatPrecisionException", "IllegalFormatArgumentIndexException", "IllegalFormatConversionException", "IllegalFormatWidthException", "IllegalFormatFlagsException", "IllegalFormatCodePointException"} {
		RegisterException(name, "IllegalFormatException")
		RegisterJavaType(TypeID("java.util."+name), "java.util.IllegalFormatException")
	}
}
