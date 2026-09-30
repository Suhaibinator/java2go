package stdjava

// RequireJavaString preserves the original nonnull String reference. Java
// call-site evaluation and helpful-null diagnostics remain compiler concerns.
func RequireJavaString(value *JavaString) *JavaString {
	return ReferenceRequireNonNull(value)
}

// JavaStringSwitchKey is an opaque host key, not text or a Java String value.
// Two big-endian bytes per UTF16 unit make the representation injective without
// losing isolated surrogates. A normal String switch rejects a null selector.
func JavaStringSwitchKey(value *JavaString) string {
	RequireJavaString(value)
	key := make([]byte, len(value.units)*2)
	for index, unit := range value.units {
		key[index*2], key[index*2+1] = byte(unit>>8), byte(unit)
	}
	return string(key)
}

// JavaStringTextOperandExecution is the text boundary used after evaluating a
// reference operand for concatenation or printing. Unlike String.valueOf(Object),
// these consumers display a null result from source toString as the text null.
// Conversion occurs once with the caller's Execution; thrown callbacks propagate.
func JavaStringTextOperandExecution(execution *Execution, value any) *JavaString {
	text := JavaStringValueOfExecution(execution, value)
	if text == nil {
		return JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'})
	}
	return text
}
