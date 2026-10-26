package stdjava_test

import (
	stdjava "github.com/NickyBoy89/java2go/stdjava"
	"strings"
	"testing"
)

var boundaryRuneSink rune

type boundaryCase struct {
	name, value string
	index       int32
	want        rune
}

func boundaryCases() []boundaryCase {
	return []boundaryCase{
		{"ascii", strings.Repeat("a", 510) + "_3", 0, 'a'},
		{"bmp", strings.Repeat("\u03bb", 510) + "_6", 0, 0x03BB},
		{"nul", strings.Repeat("a", 511) + "\x00", 511, 0},
		{"supplementary_high", strings.Repeat("\U0001f600", 255) + "_9", 0, 0xD83D},
		{"supplementary_low", strings.Repeat("\U0001f600", 255) + "_9", 1, 0xDE00},
	}
}

// This preserves the measured warmed AllocsPerRun(10) zero target.
// Its integer-truncated result is not a universal per-invocation guarantee.
func TestStringRequireNonNullValidIndexAllocations(t *testing.T) {
	for _, tc := range boundaryCases() {
		got := testing.AllocsPerRun(10, func() { boundaryRuneSink = stdjava.StringCharAt(stdjava.StringRequireNonNull(tc.value), tc.index) })
		if boundaryRuneSink != tc.want {
			t.Fatalf("boundary semantic mismatch name=%s", tc.name)
		}
		if got != 0 {
			t.Errorf("boundary valid index name=%s allocated %g, want zero", tc.name, got)
		}
	}
}
