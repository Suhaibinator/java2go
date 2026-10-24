package stdjava

import (
	"math/big"
	"math/rand"
	"slices"
	"testing"
)

func TestCampaignJoiningCapacityBigIntOracle(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	oracle := func(total, increment, count, limit int) (int, bool) {
		if total < 0 || increment < 0 || count < 0 || limit < 0 {
			return 0, false
		}
		sum := new(big.Int).Add(big.NewInt(int64(total)), new(big.Int).Mul(big.NewInt(int64(increment)), big.NewInt(int64(count))))
		if sum.Cmp(big.NewInt(int64(limit))) > 0 {
			return 0, false
		}
		return int(sum.Int64()), true
	}
	check := func(total, increment, count, limit int) {
		t.Helper()
		want, admitted := oracle(total, increment, count, limit)
		got, ok := checkedJoiningUTF16Length(total, increment, count, limit)
		if got != want || ok != admitted {
			t.Fatalf("total=%d increment=%d count=%d limit=%d: got(%d,%v) want(%d,%v)", total, increment, count, limit, got, ok, want, admitted)
		}
	}
	boundaries := []int{-1, 0, 1, 4, maxInt / 2, maxInt - 1, maxInt}
	for _, total := range boundaries {
		for _, increment := range boundaries {
			for _, count := range boundaries {
				for _, limit := range boundaries {
					check(total, increment, count, limit)
				}
			}
		}
	}
	for _, seed := range []int64{17, 41, 97} {
		rng := rand.New(rand.NewSource(seed))
		for i := 0; i < 4096; i++ {
			values := [4]int{}
			for j := range values {
				values[j] = int(rng.Uint64() & uint64(maxInt))
			}
			check(values[0], values[1], values[2], values[3])
			check(values[0]%128, values[1]%32, values[2]%16, values[3]%256)
		}
	}
}

func TestCampaignJoiningCapacityEncounterContentIdentity(t *testing.T) {
	alphabet := []*JavaString{nil, NewJavaStringUTF16(nil), NewJavaStringUTF16([]uint16{'A'}), NewJavaStringUTF16([]uint16{0xd800, 0, 0xdfff})}
	wrappers := []*JavaString{NewJavaStringUTF16(nil), NewJavaStringUTF16([]uint16{'['}), NewJavaStringUTF16([]uint16{0xdfff, 0})}
	separators := []*JavaString{NewJavaStringUTF16(nil), NewJavaStringUTF16([]uint16{'|'}), NewJavaStringUTF16([]uint16{0xd83d, 0xde00, 0})}
	for count := 0; count <= 4; count++ {
		combinations := 1
		for i := 0; i < count; i++ {
			combinations *= len(alphabet)
		}
		for code := 0; code < combinations; code++ {
			parts := make([]*JavaString, count)
			digits := code
			for i := range parts {
				parts[i] = alphabet[digits%len(alphabet)]
				digits /= len(alphabet)
			}
			for _, separator := range separators {
				for wi, prefix := range wrappers {
					suffix := wrappers[len(wrappers)-1-wi]
					want := prefix.UTF16Copy()
					for i, part := range parts {
						if i != 0 {
							want = append(want, separator.UTF16Copy()...)
						}
						if part == nil {
							want = append(want, 'n', 'u', 'l', 'l')
						} else {
							want = append(want, part.UTF16Copy()...)
						}
					}
					want = append(want, suffix.UTF16Copy()...)
					stream := NewStream(parts...)
					first := JavaStringStreamJoining(stream, separator, prefix, suffix)
					second := JavaStringStreamJoining(stream, separator, prefix, suffix)
					if first == nil || second == nil || first == second || !slices.Equal(first.UTF16Copy(), want) || !slices.Equal(second.UTF16Copy(), want) {
						t.Fatalf("count=%d code=%d wrapper=%d: content/identity", count, code, wi)
					}
					for _, part := range append(append([]*JavaString{}, parts...), separator, prefix, suffix) {
						if part != nil && (first == part || second == part) {
							t.Fatal("returned input wrapper")
						}
					}
					exported := first.UTF16Copy()
					if len(exported) != 0 {
						exported[0] ^= 0xffff
					}
					if !slices.Equal(first.UTF16Copy(), want) || !slices.Equal(second.UTF16Copy(), want) {
						t.Fatal("export mutation altered retained output")
					}
				}
			}
		}
	}
}
