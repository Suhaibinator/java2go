package stdjava

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
)

func TestMathMultiplyExactAgainstBigInteger(t *testing.T) {
	edges := []int64{math.MinInt64, math.MinInt64 + 1, math.MinInt32, -3037000500, -46341, -1, 0, 1, 46340, 46341, 3037000499, 3037000500, math.MaxInt32, math.MaxInt64}
	var pairs [][2]int64
	for _, a := range edges {
		for _, b := range edges {
			pairs = append(pairs, [2]int64{a, b})
		}
	}
	random := rand.New(rand.NewSource(17))
	for i := 0; i < 300; i++ {
		pairs = append(pairs, [2]int64{int64(random.Uint64()), int64(random.Uint64())})
	}
	for _, pair := range pairs {
		for _, bits := range []uint{32, 64} {
			a, b := pair[0], pair[1]
			if bits == 32 {
				a, b = int64(int32(a)), int64(int32(b))
			}
			want := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
			limit := new(big.Int).Lsh(big.NewInt(1), bits-1)
			min := new(big.Int).Neg(new(big.Int).Set(limit))
			max := new(big.Int).Sub(limit, big.NewInt(1))
			overflow := want.Cmp(min) < 0 || want.Cmp(max) > 0
			func() {
				defer func() {
					failure := recover()
					if overflow {
						message := "integer overflow"
						if bits == 64 {
							message = "long overflow"
						}
						if !CaughtAs(failure, "ArithmeticException") || GetMessage(failure) != message {
							t.Errorf("%d-bit %d * %d failure=%v", bits, a, b, failure)
						}
					} else if failure != nil {
						t.Errorf("%d-bit %d * %d unexpected %v", bits, a, b, failure)
					}
				}()
				var got int64
				if bits == 32 {
					got = int64(MathMultiplyExact(int32(a), int32(b)))
				} else {
					got = MathMultiplyExact(a, b)
				}
				if overflow {
					t.Errorf("%d-bit %d * %d did not throw", bits, a, b)
				} else if got != want.Int64() {
					t.Errorf("%d-bit %d * %d=%d want %s", bits, a, b, got, want)
				}
			}()
		}
	}
}
