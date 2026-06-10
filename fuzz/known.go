package fuzz

import "strings"

// knownSignatureMarkers lists substrings of normalized error/diff signatures that
// correspond to ALREADY-KNOWN, already-assigned open bugs (see e2e skipReasons
// and tasks #6/#10/#11). The fuzzer still records these to the corpus as
// regression inputs, but flags them as known so they do not drown out genuinely
// new root causes in the report.
//
// Each entry cites the owning bug. Matching is substring-based against the
// normalized Signature() string.
var knownSignatureMarkers = []struct {
	marker string
	reason string
}{
	// int locals emitted as untyped Go int rather than int32 — pervasive; causes
	// every "mismatched types int and int32" and "constant overflows int32".
	{"mismatched types int and intN", "int locals not pinned to int32 (task #10)"},
	{"overflows intN", "int locals not pinned to int32 (task #10)"},
	{"truncated to int", "int/double mixing from untyped int locals (task #10)"},
	{"as intN value", "int locals not pinned to int32 (task #10)"},
	{"as rune value", "char/rune vs int typing (task #10 int typing)"},
	{"mismatched types rune and int", "char/rune vs int typing (task #10 int typing)"},
}

// IsKnown reports whether a divergence signature matches an already-tracked open
// bug, returning the citing reason. New signatures return ("", false).
func IsKnown(sig string) (string, bool) {
	for _, k := range knownSignatureMarkers {
		if strings.Contains(sig, k.marker) {
			return k.reason, true
		}
	}
	return "", false
}
