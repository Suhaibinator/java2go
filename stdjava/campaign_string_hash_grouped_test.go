package stdjava

import (
	"fmt"
	"math/big"
	"sync"
	"testing"
)

func groupedHashBigIntOracle(units []uint16) int32 {
	h := new(big.Int)
	modulus := new(big.Int).Lsh(big.NewInt(1), 32)
	unit := new(big.Int)
	for _, v := range units {
		h.Mul(h, big.NewInt(31))
		h.Add(h, unit.SetUint64(uint64(v)))
		h.Mod(h, modulus)
	}
	return int32(uint32(h.Uint64()))
}
func groupedHashUnits(length, mode int) []uint16 {
	out := make([]uint16, length)
	state := uint32(17)
	for i := range out {
		state = state*1664525 + 1013904223
		switch mode {
		case 0:
			out[i] = uint16('a' + (state>>24)&3)
		case 1:
			out[i] = uint16(0xe000 + (state>>24)&3)
		case 2:
			out[i] = []uint16{0, 0xd800, 0xdc00, 0xffff}[state>>30]
		case 3:
			out[i] = 0
		default:
			out[i] = uint16(state >> 16)
		}
	}
	return out
}
func TestCampaignGroupedHashIndependentUTF16Oracle(t *testing.T) {
	lengths := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 1023, 1024, 1025, 8191, 8192, 8193}
	for _, n := range lengths {
		for mode := 0; mode < 5; mode++ {
			units := groupedHashUnits(n, mode)
			want := groupedHashBigIntOracle(units)
			value := NewJavaStringUTF16(units)
			copied := CopyJavaString(value)
			for i := range units {
				units[i] ^= 0xffff
			}
			for _, target := range []*JavaString{value, copied, CopyJavaString(value)} {
				first := target.HashCode()
				cached := target.HashCode()
				if first != want || cached != want {
					t.Fatalf("length%d mode%d hash=%d want%d", n, mode, target.HashCode(), want)
				}
				exported := target.UTF16Copy()
				for i := range exported {
					exported[i] = 0
				}
				if target.HashCode() != want {
					t.Fatal("exported units changed immutable hash")
				}
			}
			if copied == value || !copied.Equals(value) {
				t.Fatal("copy lost fresh identity/equal content")
			}
		}
	}
	var absent *JavaString
	assertBoxedReferencePanic(t, "NullPointerException", func() { absent.HashCode() })
}
func TestCampaignGroupedHashConcurrentFirstAndCached(t *testing.T) {
	for _, mode := range []int{2, 3, 4} {
		units := groupedHashUnits(8193, mode)
		want := groupedHashBigIntOracle(units)
		value := NewJavaStringUTF16(units)
		var wg sync.WaitGroup
		for worker := 0; worker < 8; worker++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 128; i++ {
					if value.HashCode() != want || CopyJavaString(value).HashCode() != want {
						t.Error("concurrent content hash")
					}
				}
			}()
		}
		wg.Wait()
	}
}

var groupedHashBenchValue *JavaString
var groupedHashBenchResult int32

func BenchmarkCampaignGroupedHash(b *testing.B) {
	for _, n := range []int{0, 1, 3, 4, 5, 8, 16, 128, 2048, 8192} {
		modes := []int{0}
		if n >= 128 {
			modes = []int{0, 1, 2, 3}
		}
		for _, mode := range modes {
			units := groupedHashUnits(n, mode)
			want := groupedHashBigIntOracle(units)
			b.Run(fmt.Sprintf("newFirstHash/n%d/m%d", n, mode), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					value := NewJavaStringUTF16(units)
					groupedHashBenchResult = value.HashCode()
					groupedHashBenchValue = value
				}
				if groupedHashBenchResult != want || groupedHashBenchValue.Length() != int32(n) {
					b.Fatal("cold hash/content")
				}
			})
		}
	}
	for _, n := range []int{0, 4, 2048, 8192} {
		units := groupedHashUnits(n, 2)
		value := NewJavaStringUTF16(units)
		want := groupedHashBigIntOracle(units)
		value.HashCode()
		b.Run(fmt.Sprintf("cached/n%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				groupedHashBenchResult = value.HashCode()
			}
			if groupedHashBenchResult != want {
				b.Fatal("cached hash")
			}
		})
		b.Run(fmt.Sprintf("newNoHash/n%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				groupedHashBenchValue = NewJavaStringUTF16(units)
			}
			if groupedHashBenchValue.Length() != int32(n) {
				b.Fatal("unhashed content")
			}
		})
	}
}
