package stdjava

import (
	"slices"
	"testing"
	"unicode/utf16"
)

// The primitive candidate without the reference adapter compiles this control
// and reports the same UnsupportedOperationException observed by the Java flows.
func TestNIOByteOrderCanonicalReferenceStringBoundary(t *testing.T) {
	execution := NewExecution()
	for _, entry := range []struct {
		order *ByteOrder
		label string
	}{
		{ByteOrderBIG_ENDIAN, "BIG_ENDIAN"},
		{ByteOrderLITTLE_ENDIAN, "LITTLE_ENDIAN"},
	} {
		var got *JavaString
		func() {
			defer func() {
				if thrown := recover(); thrown != nil {
					t.Errorf("canonical ByteOrder conversion panicked: %v", thrown)
				}
			}()
			got = JavaStringValueOfExecution(execution, entry.order)
		}()
		if got == nil {
			if !t.Failed() {
				t.Error("ByteOrder canonical conversion returned null")
			}
			continue
		}
		want := utf16.Encode([]rune(entry.label))
		if !slices.Equal(got.UTF16Copy(), want) {
			t.Errorf("ByteOrder text UTF16 %x, want %x", got.UTF16Copy(), want)
		}
		if got != JavaStringLiteralUTF16(want) || got != JavaStringValueOfExecution(NewExecution(), entry.order) {
			t.Error("ByteOrder.toString lost its immutable name reference")
		}
		if operand := JavaStringTextOperandExecution(execution, entry.order); operand != got {
			t.Error("ByteOrder text-operand conversion replaced the canonical name reference")
		}
		if entry.order.ToString() != entry.label || entry.order.String() != entry.label {
			t.Error("native ByteOrder label differs from its Java reference label")
		}
	}
	stringer, ok := any((*ByteOrder)(nil)).(interface {
		StringJava2goExecution(*Execution) *JavaString
	})
	if !ok {
		t.Error("ByteOrder missing canonical reference-returning StringJava2goExecution adapter")
		return
	}
	expectBoxedException(t, "NullPointerException", func() {
		stringer.StringJava2goExecution(execution)
	})
}
