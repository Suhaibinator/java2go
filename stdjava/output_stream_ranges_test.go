package stdjava

import "testing"

func TestOutputStreamRangesCopyAndExceptions(t *testing.T) {
	source := PrimitiveArrayLiteral(PrimitiveByteTypeID, int8(1), int8(-128), int8(-1), int8(4))
	output := NewByteArrayOutputStream()
	output.WriteRange(source, 1, 2)
	source.Elements[1] = 99
	output.WriteRange(source, 4, 0)
	bytes := output.ToByteArray().Elements
	if len(bytes) != 2 || bytes[0] != -128 || bytes[1] != -1 {
		t.Fatalf("range/copy mismatch %v", bytes)
	}
	assertDigestIOPanic(t, "NullPointerException", func() { output.WriteRange(nil, 0, 0) })
	assertDigestIOPanic(t, "IndexOutOfBoundsException", func() { output.WriteRange(source, -1, 1) })
	assertDigestIOPanic(t, "IndexOutOfBoundsException", func() { output.WriteRange(source, 1, 4) })
}
