package stdjava

import (
	"runtime"
	"strconv"
	"sync"
	"testing"
)

func TestCampaignIntTextOwnedFreshContract(t *testing.T) {
	values := []int32{-2147483648, -100, -1, 0, 7, 99, 100, 101, 123456789, 2147483647}
	const copies = 32
	retained := make([]*JavaString, len(values)*copies)
	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for index := w; index < len(retained); index += 4 {
				retained[index] = JavaStringValueOfInt(values[index%len(values)])
			}
		}(worker)
	}
	wg.Wait()
	seen := make(map[*JavaString]bool)
	for index, s := range retained {
		want := strconv.FormatInt(int64(values[index%len(values)]), 10)
		if s == nil || seen[s] || s.cachedHash.Load() != 0 || len(s.units) != len(want) {
			t.Fatalf("fresh initial state at %d", index)
		}
		seen[s] = true
		var hash uint32
		for i := range want {
			if s.units[i] != uint16(want[i]) {
				t.Fatalf("units at %d", index)
			}
			hash = 31*hash + uint32(s.units[i])
		}
		firstHash := s.HashCode()
		if firstHash != int32(hash) {
			t.Fatalf("first hash at %d", index)
		}
		repeatedHash := s.HashCode()
		if repeatedHash != firstHash {
			t.Fatalf("cached hash at %d", index)
		}
		copy := s.UTF16Copy()
		copy[0] ^= 1
		if s.units[0] != uint16(want[0]) {
			t.Fatalf("exposed units at %d", index)
		}
	}
	// White-box mutation proves independently owned backing for equal text.
	first, second := JavaStringValueOfInt(100), JavaStringValueOfInt(100)
	first.units[0] = '9'
	if second.units[0] != '1' {
		t.Fatal("cross-call backing alias")
	}
	if JavaStringValueOfInt(0) == nil {
		t.Fatal("primitive conversion produced null")
	}
	runtime.KeepAlive(retained)
}

func TestCampaignIntTextExactlyOwnedAllocations(t *testing.T) {
	for _, v := range []int32{-2147483648, -1, 0, 7, 99, 100, 123456789, 2147483647} {
		got := testing.AllocsPerRun(1000, func() { campaignPublic140IntTextSink = JavaStringValueOfInt(v) })
		// Two escaping objects are required: fresh String and owned UTF16 backing.
		if got != 2 {
			t.Errorf("value=%d: observed %g allocations, want exactly 2 owned allocations", v, got)
		}
	}
	runtime.KeepAlive(campaignPublic140IntTextSink)
}
