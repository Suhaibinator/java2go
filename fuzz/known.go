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
	// every int/int32 mismatch, int32 overflow, and int32-target assignment error.
	{"mismatched-types:int+int32", "int locals not pinned to int32 (task #10)"},
	{"mismatched-types:int32+int", "int locals not pinned to int32 (task #10)"},
	{"overflows:int32", "int locals not pinned to int32 (task #10)"},
	{"truncated:int", "int/double mixing from untyped int locals (task #10)"},
	{"cannot-use-as:int32", "int locals not pinned to int32 (task #10)"},
	{"cannot-use-as:int", "int locals not pinned to int32 (task #10)"},
	// char (rune) vs int promotion not handled — char arithmetic stays rune.
	{"cannot-use-as:rune", "char/rune vs int typing (task #10 int typing)"},
	{"overflows:rune", "char/rune vs int typing (task #10 int typing)"},
	{"truncated:rune", "char/rune vs int typing (task #10 int typing)"},
	{"mismatched-types:rune+int", "char/rune vs int typing (task #10 int typing)"},
	{"mismatched-types:int+rune", "char/rune vs int typing (task #10 int typing)"},
	// int local mixed with double — variant of the int-typing gap.
	{"mismatched-types:int+float64", "int locals not pinned, mixed with float64 (task #10)"},
	{"mismatched-types:float64+int", "int locals not pinned, mixed with float64 (task #10)"},

	// --- NEW fuzzer-found root causes (reported to team-lead; task #14) ---
	// >>>= compound assign emits an undefined, non-assigning Go call.
	{"undefined: V", "fuzzer K-bug: >>>= emits undefined UnsignedRightShiftAssignment (task #14)"},
	// Unused local emitted without `_ = v`; valid Java, invalid Go.
	{"declared and not used", "fuzzer K-bug: unused local not discarded (task #14)"},
	// -9223372036854775808L emitted as -int64(9223372036854775808); overflows.
	{"overflows:int64", "fuzzer K-bug: negative long-min literal mis-emitted (task #14)"},
	// Explicit upcast (Super) sub emits a failing Go type assertion.
	{"interface conversion", "fuzzer K-bug: upcast emits failing type assertion (task #14)"},
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

// knownOpenCorpus marks specific corpus files (by basename, no extension) whose
// underlying bug is still open but whose OUTPUT_MISMATCH signature shape is too
// generic to safely list in knownSignatureMarkers (it could collide with a
// future genuine mismatch). The replay test skips these by name, exactly as the
// e2e suite skips a fixture by key. Remove an entry once its bug is fixed to turn
// the corpus program into an enforced regression test.
var knownOpenCorpus = map[string]string{
	"DoubleFormat":   "fuzzer K-bug: integral double prints '96' not '96.0' (System.out.println(double); task #14)",
	"StringPlusChar": "fuzzer K-bug: string + char concatenates the numeric code, not the glyph (task #14)",
}

// IsKnownOpenCorpus reports whether a corpus file (by stem) is a still-open bug
// tracked outside the signature list.
func IsKnownOpenCorpus(stem string) (string, bool) {
	r, ok := knownOpenCorpus[stem]
	return r, ok
}
