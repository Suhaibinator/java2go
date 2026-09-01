package stdjava

import (
	"math"
	"testing"
)

func TestNumberAccessorsAndDoubleValueOf(t *testing.T) {
	if got := NumberDoubleValue(int32(7)); got != 7 {
		t.Fatalf("NumberDoubleValue(int32(7)) = %v, want 7", got)
	}
	if got := NumberFloatValue(float64(1.25)); got != float32(1.25) {
		t.Fatalf("NumberFloatValue(1.25) = %v, want 1.25", got)
	}
	if got := NumberIntValue(math.NaN()); got != 0 {
		t.Fatalf("NumberIntValue(NaN) = %v, want 0", got)
	}
	if got := NumberIntValue(math.Inf(1)); got != math.MaxInt32 {
		t.Fatalf("NumberIntValue(+Inf) = %v, want %v", got, int32(math.MaxInt32))
	}
	if got := NumberByteValue(int32(130)); got != -126 {
		t.Fatalf("NumberByteValue(130) = %v, want -126", got)
	}
	if got := DoubleValueOf("2.5"); got != 2.5 {
		t.Fatalf("DoubleValueOf(string) = %v, want 2.5", got)
	}
	if got := DoubleValueOf(float32(3.5)); got != 3.5 {
		t.Fatalf("DoubleValueOf(float32) = %v, want 3.5", got)
	}
}

func TestNumberAccessorRejectsNull(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("NumberDoubleValue(nil) did not panic")
		}
		_, ok := recovered.(NullPointerException)
		if !ok {
			t.Fatalf("NumberDoubleValue(nil) panic = %T, want NullPointerException", recovered)
		}
	}()
	NumberDoubleValue(nil)
}
