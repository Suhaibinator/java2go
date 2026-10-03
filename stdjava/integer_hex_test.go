package stdjava

import "testing"

func TestIntegerToHexString(t *testing.T) {
	for _, test := range []struct {
		value    int32
		expected string
	}{
		{0, "0"}, {42, "2a"}, {-1, "ffffffff"}, {-2147483648, "80000000"}, {2147483647, "7fffffff"},
	} {
		if got := IntegerToHexString(test.value); got != test.expected {
			t.Errorf("IntegerToHexString(%d) = %q; want %q", test.value, got, test.expected)
		}
	}
}
