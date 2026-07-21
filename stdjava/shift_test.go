package stdjava

import "testing"

// Examples from the JavaScript reference at https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Unsigned_right_shift

func TestRightShiftPositive(t *testing.T) {
	if UnsignedRightShift(9, 2) != 2 {
		t.Errorf("Shifted 9 >>> 2. Expected 2 but got %d", UnsignedRightShift(9, 2))
	}
}

func TestRightShiftNegative(t *testing.T) {
	if UnsignedRightShift(-9, 2) != 1073741821 {
		t.Errorf("Shifted -9 >>> 2. Expected 1073741821 but got %d", UnsignedRightShift(-9, 2))
	}
}

func TestRightShiftLongPreserves64BitWidth(t *testing.T) {
	if got := UnsignedRightShift(int64(-1), 1); got != 9223372036854775807 {
		t.Fatalf("long -1 >>> 1 = %d", got)
	}
}

func TestRightShiftAssignmentUsesProvidedAmountAndStores(t *testing.T) {
	value := int32(-8)
	UnsignedRightShiftAssignment(&value, 1)
	if value != 2147483644 {
		t.Fatalf("int -8 >>>= 1 = %d", value)
	}
}
