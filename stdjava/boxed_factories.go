package stdjava

// ValueOfString helpers retain the String overload signatures explicitly.
// Primitive valueOf calls can use BoxX directly; constructors use NewX after
// parsing, which deliberately bypasses caches.
func BooleanValueOfString(value string) *Boolean {
	return BoxBoolean(ParseBoolean(value))
}
func ByteValueOfString(value string, radix ...int32) *Byte {
	return BoxByte(ParseByte(value, radix...))
}
func ShortValueOfString(value string, radix ...int32) *Short {
	return BoxShort(ParseShort(value, radix...))
}
func IntegerValueOfString(value string, radix ...int32) *Integer {
	return BoxInteger(ParseInt(value, radix...))
}
func LongValueOfString(value string, radix ...int32) *Long {
	return BoxLong(ParseLong(value, radix...))
}
func FloatValueOfString(value string) *Float {
	return BoxFloat(ParseFloat(value))
}
func DoubleValueOfString(value string) *Double {
	return BoxDouble(ParseDouble(value))
}

// The erased dispatchers serve intrinsic paths before argument signatures are
// fully specialized. They permit only the primitive widening conversions of
// the Java overload, plus unboxing followed by primitive widening. The compiler
// still rejects invalid Java invocations before emitting a call.
func BooleanValueOf(argument any, radix ...int32) *Boolean {
	if argument == nil {
		return BooleanValueOfString(NullString())
	}
	if text, ok := argument.(string); ok {
		if len(radix) != 0 {
			panic(NewIllegalArgumentException("unexpected radix"))
		}
		return BooleanValueOfString(text)
	}
	if len(radix) != 0 {
		panic(NewIllegalArgumentException("radix requires String argument"))
	}
	switch value := argument.(type) {
	case bool:
		return BoxBoolean(bool(value))
	case *Boolean:
		return BoxBoolean(bool(UnboxBoolean(value)))
	default:
		panic(NewClassCastException("invalid Boolean.valueOf argument"))
	}
}
func ByteValueOf(argument any, radix ...int32) *Byte {
	if argument == nil {
		return ByteValueOfString(NullString(), radix...)
	}
	if text, ok := argument.(string); ok {
		return ByteValueOfString(text, radix...)
	}
	if len(radix) != 0 {
		panic(NewIllegalArgumentException("radix requires String argument"))
	}
	switch value := argument.(type) {
	case int8:
		return BoxByte(int8(value))
	case *Byte:
		return BoxByte(int8(UnboxByte(value)))
	default:
		panic(NewClassCastException("invalid Byte.valueOf argument"))
	}
}
func ShortValueOf(argument any, radix ...int32) *Short {
	if argument == nil {
		return ShortValueOfString(NullString(), radix...)
	}
	if text, ok := argument.(string); ok {
		return ShortValueOfString(text, radix...)
	}
	if len(radix) != 0 {
		panic(NewIllegalArgumentException("radix requires String argument"))
	}
	switch value := argument.(type) {
	case int8:
		return BoxShort(int16(value))
	case int16:
		return BoxShort(int16(value))
	case *Byte:
		return BoxShort(int16(UnboxByte(value)))
	case *Short:
		return BoxShort(int16(UnboxShort(value)))
	default:
		panic(NewClassCastException("invalid Short.valueOf argument"))
	}
}
func CharacterValueOf(argument any, radix ...int32) *Character {
	if len(radix) != 0 {
		panic(NewIllegalArgumentException("radix requires String argument"))
	}
	switch value := argument.(type) {
	case int32:
		return BoxCharacter(rune(value))
	case *Character:
		return BoxCharacter(rune(UnboxCharacter(value)))
	default:
		panic(NewClassCastException("invalid Character.valueOf argument"))
	}
}
func IntegerValueOf(argument any, radix ...int32) *Integer {
	if argument == nil {
		return IntegerValueOfString(NullString(), radix...)
	}
	if text, ok := argument.(string); ok {
		return IntegerValueOfString(text, radix...)
	}
	if len(radix) != 0 {
		panic(NewIllegalArgumentException("radix requires String argument"))
	}
	switch value := argument.(type) {
	case int8:
		return BoxInteger(int32(value))
	case int16:
		return BoxInteger(int32(value))
	case int32:
		return BoxInteger(int32(value))
	case *Byte:
		return BoxInteger(int32(UnboxByte(value)))
	case *Short:
		return BoxInteger(int32(UnboxShort(value)))
	case *Character:
		return BoxInteger(int32(UnboxCharacter(value)))
	case *Integer:
		return BoxInteger(int32(UnboxInteger(value)))
	default:
		panic(NewClassCastException("invalid Integer.valueOf argument"))
	}
}
func LongValueOf(argument any, radix ...int32) *Long {
	if argument == nil {
		return LongValueOfString(NullString(), radix...)
	}
	if text, ok := argument.(string); ok {
		return LongValueOfString(text, radix...)
	}
	if len(radix) != 0 {
		panic(NewIllegalArgumentException("radix requires String argument"))
	}
	switch value := argument.(type) {
	case int8:
		return BoxLong(int64(value))
	case int16:
		return BoxLong(int64(value))
	case int32:
		return BoxLong(int64(value))
	case int64:
		return BoxLong(int64(value))
	case *Byte:
		return BoxLong(int64(UnboxByte(value)))
	case *Short:
		return BoxLong(int64(UnboxShort(value)))
	case *Character:
		return BoxLong(int64(UnboxCharacter(value)))
	case *Integer:
		return BoxLong(int64(UnboxInteger(value)))
	case *Long:
		return BoxLong(int64(UnboxLong(value)))
	default:
		panic(NewClassCastException("invalid Long.valueOf argument"))
	}
}
func FloatValueOf(argument any, radix ...int32) *Float {
	if argument == nil {
		return FloatValueOfString(NullString())
	}
	if text, ok := argument.(string); ok {
		if len(radix) != 0 {
			panic(NewIllegalArgumentException("unexpected radix"))
		}
		return FloatValueOfString(text)
	}
	if len(radix) != 0 {
		panic(NewIllegalArgumentException("radix requires String argument"))
	}
	switch value := argument.(type) {
	case int8:
		return BoxFloat(float32(value))
	case int16:
		return BoxFloat(float32(value))
	case int32:
		return BoxFloat(float32(value))
	case int64:
		return BoxFloat(float32(value))
	case float32:
		return BoxFloat(float32(value))
	case *Byte:
		return BoxFloat(float32(UnboxByte(value)))
	case *Short:
		return BoxFloat(float32(UnboxShort(value)))
	case *Character:
		return BoxFloat(float32(UnboxCharacter(value)))
	case *Integer:
		return BoxFloat(float32(UnboxInteger(value)))
	case *Long:
		return BoxFloat(float32(UnboxLong(value)))
	case *Float:
		return BoxFloat(float32(UnboxFloat(value)))
	default:
		panic(NewClassCastException("invalid Float.valueOf argument"))
	}
}
func DoubleValueOf(argument any, radix ...int32) *Double {
	if argument == nil {
		return DoubleValueOfString(NullString())
	}
	if text, ok := argument.(string); ok {
		if len(radix) != 0 {
			panic(NewIllegalArgumentException("unexpected radix"))
		}
		return DoubleValueOfString(text)
	}
	if len(radix) != 0 {
		panic(NewIllegalArgumentException("radix requires String argument"))
	}
	switch value := argument.(type) {
	case int8:
		return BoxDouble(float64(value))
	case int16:
		return BoxDouble(float64(value))
	case int32:
		return BoxDouble(float64(value))
	case int64:
		return BoxDouble(float64(value))
	case float32:
		return BoxDouble(float64(value))
	case float64:
		return BoxDouble(float64(value))
	case *Byte:
		return BoxDouble(float64(UnboxByte(value)))
	case *Short:
		return BoxDouble(float64(UnboxShort(value)))
	case *Character:
		return BoxDouble(float64(UnboxCharacter(value)))
	case *Integer:
		return BoxDouble(float64(UnboxInteger(value)))
	case *Long:
		return BoxDouble(float64(UnboxLong(value)))
	case *Float:
		return BoxDouble(float64(UnboxFloat(value)))
	case *Double:
		return BoxDouble(float64(UnboxDouble(value)))
	default:
		panic(NewClassCastException("invalid Double.valueOf argument"))
	}
}

// Float's double constructor is a Java-sanctioned narrowing conversion;
// Float.valueOf(float) deliberately does not accept a double.
func NewFloatFromDouble(value float64) *Float { return NewFloat(float32(value)) }
