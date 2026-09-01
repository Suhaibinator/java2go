package stdjava

import (
	"math"
	"testing"
)

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
func TestJavaMathTrigMatchesStrictMathRecurrence(t *testing.T) {
	x, y, total := 0.5, 0.8, 0.0
	for range 10 {
		nextX := JavaMathSin(x*math.Pi) - JavaMathCos(y*math.Pi)
		nextY := JavaMathCos(x*math.Pi) + JavaMathSin(y*math.Pi)
		total += math.Sqrt(math.Pow(nextX-x, 2) + math.Pow(nextY-y, 2))
		x, y = nextX, nextY
	}
	const wantBits = uint64(0x403552e8355d917d)
	if got := math.Float64bits(total); got != wantBits {
		t.Fatalf("recurrence bits = %016x (%v), want %016x", got, total, wantBits)
	}
}

func TestJavaMathTrigSpecialCases(t *testing.T) {
	negativeZero := math.Copysign(0, -1)
	if got := math.Float64bits(JavaMathSin(negativeZero)); got != math.Float64bits(negativeZero) {
		t.Fatalf("JavaMathSin(-0) bits = %016x, want negative zero", got)
	}
	if got := JavaMathCos(negativeZero); got != 1 {
		t.Fatalf("JavaMathCos(-0) = %v, want 1", got)
	}
}
