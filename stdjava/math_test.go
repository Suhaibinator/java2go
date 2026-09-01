package stdjava

import "testing"

func TestMathAbsMaxMin(t *testing.T) {
	if got := MathAbs(int32(-5)); got != 5 {
		t.Errorf("MathAbs(int32) = %d, want 5", got)
	}
	if got := MathAbs(-2.5); got != 2.5 {
		t.Errorf("MathAbs(float64) = %v, want 2.5", got)
	}
	if got := MathMax(int32(3), int32(7)); got != 7 {
		t.Errorf("MathMax = %d, want 7", got)
	}
	if got := MathMin(int32(3), int32(7)); got != 3 {
		t.Errorf("MathMin = %d, want 3", got)
	}
}

func TestMathRound(t *testing.T) {
	// Java rounds half up (toward positive infinity).
	cases := map[float64]int64{2.5: 3, -2.5: -2, 2.4: 2, 2.6: 3}
	for in, want := range cases {
		if got := MathRound(in); got != want {
			t.Errorf("MathRound(%v) = %d, want %d", in, got, want)
		}
	}
}
