package stdjava

import "testing"

func TestCampaignStringCharAtBoundsRuntime(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		index   int32
		message string
	}{
		{"negative", "ab", -1, "Index -1 out of bounds for length 2"},
		{"equal", "ab", 2, "Index 2 out of bounds for length 2"},
		{"greater", "ab", 8, "Index 8 out of bounds for length 2"},
		{"empty", "", 0, "Index 0 out of bounds for length 0"},
		{"supplementary", "😀", 2, "Index 2 out of bounds for length 2"},
		{"minimum", "ab", -2147483648, "Index -2147483648 out of bounds for length 2"},
		{"maximum", "ab", 2147483647, "Index 2147483647 out of bounds for length 2"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				recovered := recover()
				exception, ok := recovered.(StringIndexOutOfBoundsException)
				if !ok {
					t.Fatalf("panic=%T (%v), want StringIndexOutOfBoundsException", recovered, recovered)
				}
				if got := GetMessage(exception); got != test.message {
					t.Fatalf("message=%q, want %q", got, test.message)
				}
			}()
			StringCharAt(test.text, test.index)
			t.Fatal("invalid index returned without throwing")
		})
	}
	if high, low := StringCharAt("😀", 0), StringCharAt("😀", 1); high != 55357 || low != 56832 {
		t.Fatalf("surrogate units=%d,%d", high, low)
	}
}
