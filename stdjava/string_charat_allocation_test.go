package stdjava_test

import (
	"strings"
	"testing"

	stdjava "github.com/NickyBoy89/java2go/stdjava"
)

var charAtAllocationSink rune

func TestStringCharAtValidIndexAllocations(t *testing.T) {
	// Preserve the exact 512-unit inputs of the measured generated-workload guard.
	// All value construction stays outside allocation closures.
	cases := []struct {
		name  string
		value string
		index int32
		want  rune
	}{
		{"ascii", strings.Repeat("a", 510) + "_3", 0, 'a'},
		{"bmp", strings.Repeat("\u03bb", 510) + "_6", 0, 0x03BB},
		{"nul", strings.Repeat("a", 511) + "\x00", 511, 0},
		{"supplementary_high", strings.Repeat("\U0001f600", 255) + "_9", 0, 0xD83D},
		{"supplementary_low", strings.Repeat("\U0001f600", 255) + "_9", 1, 0xDE00},
	}
	for _, tc := range cases {
		got := testing.AllocsPerRun(10, func() {
			charAtAllocationSink = stdjava.StringCharAt(tc.value, tc.index)
		})
		if charAtAllocationSink != tc.want {
			t.Fatalf("helper semantic mismatch name=%s got=%d want=%d", tc.name, charAtAllocationSink, tc.want)
		}
		if got != 0 {
			t.Errorf("helper valid index name=%s allocated %g, want zero", tc.name, got)
		}
	}
}
