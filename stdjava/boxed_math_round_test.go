package stdjava

import (
	"math"
	"testing"
)

func TestMathRoundDoubleBoundaries(t *testing.T) {
	tests := []struct {
		value float64
		want  int64
	}{
		{0, 0}, {math.Copysign(0, -1), 0},
		{0.4, 0}, {0.5, 1}, {1.5, 2}, {1.75, 2},
		{-0.4, 0}, {-0.5, 0}, {-1.5, -1}, {-1.75, -2},
		{math.Nextafter(0.5, 0), 0}, {math.Nextafter(0.5, 1), 1},
		{math.Nextafter(-0.5, -1), -1}, {math.Nextafter(-0.5, 0), 0},
		{1<<52 + 1, 1<<52 + 1},
		{math.NaN(), 0}, {math.Inf(1), math.MaxInt64}, {math.Inf(-1), math.MinInt64},
		{float64(math.MaxInt64), math.MaxInt64}, {float64(math.MinInt64), math.MinInt64},
		{math.Nextafter(float64(math.MaxInt64), 0), math.MaxInt64 - 1023},
		{math.Nextafter(float64(math.MinInt64), 0), math.MinInt64 + 1024},
		{math.Nextafter(float64(math.MaxInt64), math.Inf(1)), math.MaxInt64},
		{math.Nextafter(float64(math.MinInt64), math.Inf(-1)), math.MinInt64},
	}
	for _, test := range tests {
		if got := MathRound(test.value); got != test.want {
			t.Errorf("MathRound(%g) = %d, want %d", test.value, got, test.want)
		}
	}
}

func TestMathRoundFloatBoundaries(t *testing.T) {
	tests := []struct {
		value float32
		want  int32
	}{
		{0, 0}, {float32(math.Copysign(0, -1)), 0},
		{0.4, 0}, {0.5, 1}, {1.5, 2}, {1.75, 2},
		{-0.4, 0}, {-0.5, 0}, {-1.5, -1}, {-1.75, -2},
		{math.Nextafter32(0.5, 0), 0}, {math.Nextafter32(0.5, 1), 1},
		{math.Nextafter32(-0.5, -1), -1}, {math.Nextafter32(-0.5, 0), 0},
		{1<<23 + 1, 1<<23 + 1},
		{float32(math.NaN()), 0}, {float32(math.Inf(1)), math.MaxInt32}, {float32(math.Inf(-1)), math.MinInt32},
		{float32(math.MaxInt32), math.MaxInt32}, {float32(math.MinInt32), math.MinInt32},
		{math.Nextafter32(float32(math.MaxInt32), 0), math.MaxInt32 - 127},
		{math.Nextafter32(float32(math.MinInt32), 0), math.MinInt32 + 128},
		{math.Nextafter32(float32(math.MaxInt32), float32(math.Inf(1))), math.MaxInt32},
		{math.Nextafter32(float32(math.MinInt32), float32(math.Inf(-1))), math.MinInt32},
	}
	for _, test := range tests {
		if got := MathRoundFloat(test.value); got != test.want {
			t.Errorf("MathRoundFloat(%g) = %d, want %d", test.value, got, test.want)
		}
	}
}
