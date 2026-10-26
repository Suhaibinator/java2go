package stdjava

import (
	"reflect"
	"testing"
)

func TestBooleanCanonicalParseNative(t *testing.T) {
	if JavaBooleanParseBoolean(nil) {
		t.Fatal("null must be false")
	}
	for mask := 0; mask < 16; mask++ {
		units := []uint16{'t', 'r', 'u', 'e'}
		for i := range units {
			if mask&(1<<i) != 0 {
				units[i] -= 'a' - 'A'
			}
		}
		value := NewJavaStringUTF16(units)
		if !JavaBooleanParseBoolean(value) {
			t.Fatalf("case mask %d", mask)
		}
		if !reflect.DeepEqual(value.UTF16Copy(), units) {
			t.Fatal("parser mutated units")
		}
	}
	for _, units := range [][]uint16{nil, {}, {' ', 't', 'r', 'u', 'e'}, {'t', 'r', 'u', 'e', ' '}, {'t', 'r', 0xD800, 'e'}, {'t', 'r', 0xDC00, 'e'}, {'t', 'r', 0, 'u', 'e'}, {'t', 'r', 'u', 0xFFFD}, {'t', 'r', 0x200B, 'e'}, {0xFF34, 0xFF32, 0xFF35, 0xFF25}, {'t', 'r', 'u', 'e', 0x00A0}} {
		value := NewJavaStringUTF16(units)
		if JavaBooleanParseBoolean(value) {
			t.Fatalf("unexpected true units %v", units)
		}
	}
	if !ParseBoolean("TrUe") || ParseBoolean(" true ") || ParseBoolean(NullString()) {
		t.Fatal("legacy host parser ABI")
	}
}
