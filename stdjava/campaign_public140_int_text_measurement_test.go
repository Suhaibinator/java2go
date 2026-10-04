package stdjava

import (
	"fmt"
	"runtime"
	"strconv"
	"testing"
)

var campaignPublic140IntTextSink *JavaString
var campaignPublic140IntValues = []int32{-2147483648, -1, 0, 7, 99, 100, 123456789, 2147483647}

func TestCampaignPublic140IntTextBaseline(t *testing.T) {
	retained := make([]*JavaString, 0, 2*len(campaignPublic140IntValues))
	for _, v := range campaignPublic140IntValues {
		a, b := JavaStringValueOfInt(v), JavaStringValueOfInt(v)
		if a == nil || b == nil || a == b {
			t.Fatalf("identity %d", v)
		}
		want := strconv.FormatInt(int64(v), 10)
		if len(a.units) != len(want) || a.cachedHash.Load() != 0 {
			t.Fatalf("initial state %d", v)
		}
		for i := range want {
			if a.units[i] != uint16(want[i]) {
				t.Fatalf("units %d", v)
			}
		}
		copied := a.UTF16Copy()
		copied[0] ^= 1
		if a.units[0] != uint16(want[0]) {
			t.Fatalf("copy alias %d", v)
		}
		var expected uint32
		for _, u := range a.units {
			expected = 31*expected + uint32(u)
		}
		if a.HashCode() != int32(expected) || b.cachedHash.Load() != 0 {
			t.Fatalf("hash %d", v)
		}
		retained = append(retained, a, b)
		allocs := testing.AllocsPerRun(1000, func() { campaignPublic140IntTextSink = JavaStringValueOfInt(v) })
		t.Logf("value=%d escaping_allocs_per_call=%g", v, allocs)
	}
	runtime.KeepAlive(retained)
}

func BenchmarkCampaignPublic140IntTextBaseline(b *testing.B) {
	for _, v := range campaignPublic140IntValues {
		b.Run(fmt.Sprint(v), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				campaignPublic140IntTextSink = JavaStringValueOfInt(v)
			}
			runtime.KeepAlive(campaignPublic140IntTextSink)
		})
	}
}
