package stdjava

// javaStringValueOfNativeExecution handles the final JDK wrapper classes at
// the canonical Java String boundary. The caller handles Java null, String
// identity and exact source virtual dispatch before trying this adapter.
// found is true only for a supported nonnull wrapper, never for a null result.
// Unknown native/source values are declined without invoking their methods.
func javaStringValueOfNativeExecution(_ *Execution, value any) (*JavaString, bool) {
	if javaReferenceIsNull(value) {
		return nil, false
	}
	switch boxed := value.(type) {
	case *Boolean:
		return JavaStringValueOfBoolean(boxed.BooleanValue()), true
	case *Byte:
		return JavaStringValueOfInt(int32(boxed.ByteValue())), true
	case *Short:
		return JavaStringValueOfInt(int32(boxed.ShortValue())), true
	case *Character:
		// Character holds one UTF16 code unit, which need not be a Unicode scalar.
		return JavaStringValueOfChar(boxed.CharValue()), true
	case *Integer:
		return JavaStringValueOfInt(boxed.IntValue()), true
	case *Long:
		return JavaStringValueOfLong(boxed.LongValue()), true
	case *Float:
		return JavaStringValueOfFloat(boxed.FloatValue()), true
	case *Double:
		return JavaStringValueOfDouble(boxed.DoubleValue()), true
	default:
		return nil, false
	}
}
