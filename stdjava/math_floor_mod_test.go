package stdjava

import (
	"math/big"
	"testing"
)

func TestMathFloorModFixedWidths(t *testing.T) {
	intResult := func(value int32) int32 { return value }
	for _, row := range []struct{ x, y, want int32 }{
		{16, 3, 1}, {-16, 3, 2}, {16, -3, -2}, {-16, -3, -1}, {0, 3, 0},
		{-2147483648, -1, 0}, {-2147483648, 3, 1}, {2147483647, -2147483648, -1}, {-2147483648, -2147483648, 0},
	} {
		if got := MathFloorModInt(row.x, row.y); got != row.want {
			t.Fatalf("int(%d,%d)=%d want%d", row.x, row.y, got, row.want)
		}
	}
	for _, row := range []struct {
		x       int64
		y, want int32
	}{
		{9007199254740993, 7, 5}, {-9223372036854775808, 3, 1}, {9223372036854775807, -2147483648, -1},
		{-9223372036854775808, -1, 0}, {-1, -2147483648, -1}, {1, -2147483648, -2147483647}, {9223372036854775807, 2147483647, 1},
	} {
		got := intResult(MathFloorModLongInt(row.x, row.y))
		if got != row.want {
			t.Fatalf("long,int(%d,%d)=%d want%d", row.x, row.y, got, row.want)
		}
	}
	for _, row := range []struct{ x, y, want int64 }{
		{16, 3, 1}, {-16, 3, 2}, {16, -3, -2}, {-16, -3, -1}, {9007199254740993, 7, 5},
		{-9223372036854775808, -1, 0}, {9223372036854775807, -9223372036854775808, -1}, {1, -9223372036854775808, -9223372036854775807}, {-9223372036854775808, -9223372036854775808, 0},
	} {
		if got := MathFloorModLong(row.x, row.y); got != row.want {
			t.Fatalf("long(%d,%d)=%d want%d", row.x, row.y, got, row.want)
		}
	}
}

func TestMathFloorModZeroDivisor(t *testing.T) {
	for _, row := range []struct {
		name string
		call func()
	}{
		{"int", func() { MathFloorModInt(1, 0) }}, {"int-zero", func() { MathFloorModInt(0, 0) }},
		{"long-int", func() { MathFloorModLongInt(-9223372036854775808, 0) }}, {"long-int-zero", func() { MathFloorModLongInt(0, 0) }},
		{"long", func() { MathFloorModLong(-9223372036854775808, 0) }}, {"long-zero", func() { MathFloorModLong(0, 0) }},
	} {
		t.Run(row.name, func(t *testing.T) {
			defer func() {
				failure := recover()
				if !CaughtAs(failure, "ArithmeticException") || GetMessage(failure) != "/ by zero" {
					t.Fatalf("got%v wantJavaArithmeticException / by zero", failure)
				}
			}()
			row.call()
			t.Fatal("zero divisor did not throw")
		})
	}
}

func TestMathFloorModArbitraryPrecisionCongruence(t *testing.T) {
	// An independent arbitrary-precision Euclidean modulus checks sign, width,
	// overflow boundaries and congruence without duplicating signed Go % logic.
	values := []int64{-9223372036854775808, -9223372036854775807, -9007199254740993, -2147483649, -2147483648, -65537, -17, -1, 0, 1, 16, 65537, 2147483647, 2147483648, 9007199254740993, 9223372036854775807}
	divisors := []int64{-9223372036854775808, -2147483648, -65537, -17, -3, -1, 1, 3, 17, 65537, 2147483647, 9223372036854775807}
	for _, x := range values {
		for _, y := range divisors {
			modulus := new(big.Int).Abs(big.NewInt(y))
			want := new(big.Int).Mod(big.NewInt(x), modulus)
			if y < 0 && want.Sign() != 0 {
				want.Sub(want, modulus)
			}
			if got := MathFloorModLong(x, y); got != want.Int64() {
				t.Fatalf("long(%d,%d)=%d exact=%s", x, y, got, want)
			}
			if y >= -2147483648 && y <= 2147483647 {
				if got := MathFloorModLongInt(x, int32(y)); int64(got) != want.Int64() {
					t.Fatalf("long,int(%d,%d)=%d exact=%s", x, y, got, want)
				}
				if x >= -2147483648 && x <= 2147483647 {
					if got := MathFloorModInt(int32(x), int32(y)); int64(got) != want.Int64() {
						t.Fatalf("int(%d,%d)=%d exact=%s", x, y, got, want)
					}
				}
			}
		}
	}
}
