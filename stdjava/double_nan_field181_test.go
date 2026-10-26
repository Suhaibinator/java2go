package stdjava

import (
	"math"
	"testing"
)

// OpenJDK 21 java.base.jmod ConstantValue entries: Double.NaN cp189 and
// Float.NaN cp158. Raw bits distinguish Java's constant from Go math.NaN.
func TestDoubleNaN181JDKConstantValue(t *testing.T) {
	if got := uint64(DoubleToRawLongBits(DoubleNaN())); got != 0x7ff8000000000000 {
		t.Fatalf("JDK21 Double.NaN ConstantValue bits: got %016x want 7ff8000000000000", got)
	}
}
func TestDoubleNaN181FloatUnchangedControl(t *testing.T) {
	if got := uint32(FloatToRawIntBits(FloatNaN())); got != 0x7fc00000 {
		t.Fatalf("JDK21 Float.NaN ConstantValue bits: got %08x want 7fc00000", got)
	}
}
func TestDoubleNaN181NoncanonicalOperandPayloads(t *testing.T) {
	for _, bits := range []uint64{0x7ff8000000000042, 0xfff8000000000042} {
		value := math.Float64frombits(bits)
		values := []float64{value, MathMax(value, 1.0), MathMax(1.0, value), MathMin(value, 1.0), MathMin(1.0, value)}
		for i, got := range values {
			if actual := uint64(DoubleToRawLongBits(got)); actual != bits {
				t.Fatalf("raw conversion/Math operation %d changed operand payload: got %016x want %016x", i, actual, bits)
			}
		}
	}
}
