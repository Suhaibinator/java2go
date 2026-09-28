package stdjava

import "testing"

func TestStringSplitRejectsUnsupportedJavaRegex(t *testing.T) {
	for _, pattern := range []string{`(?=b)`, `(a)\1`, `a++`} {
		t.Run(pattern, func(t *testing.T) {
			defer func() {
				if failure := recover(); !CaughtAs(failure, "UnsupportedOperationException") {
					t.Fatalf("unsupported expression %q silently accepted or wrong failure: %v", pattern, failure)
				}
			}()
			StringSplit("aab", pattern, -1)
		})
	}
}
