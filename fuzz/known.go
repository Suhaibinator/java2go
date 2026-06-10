package fuzz

import "strings"

// knownSignatureMarkers lists substrings of normalized error/diff signatures that
// correspond to ALREADY-KNOWN open bugs catalogued in
// testfiles/e2e/KNOWN_ISSUES.md (the dedup source of truth, K1-K15). The fuzzer
// still records these to the corpus as regression inputs, but flags them as known
// so they do not drown out genuinely new root causes in the report.
//
// Each entry cites the catalogued K-id. Matching is substring-based against the
// normalized Signature() string. Keep this in sync with KNOWN_ISSUES.md.
var knownSignatureMarkers = []struct {
	marker string
	reason string
}{
	// K1: int locals emitted as untyped Go int rather than int32 — pervasive;
	// causes int/int32 mismatch, int32 overflow, int32-target assignment errors.
	{"mismatched-types:int+int32", "K1 int locals not pinned to int32 (ROADMAP §6, task #10/#15)"},
	{"mismatched-types:int32+int", "K1 int locals not pinned to int32 (ROADMAP §6, task #10/#15)"},
	{"overflows:int32", "K1 int locals not pinned to int32 (ROADMAP §6, task #10/#15)"},
	{"truncated:int", "K1 int/double mixing from untyped int locals (ROADMAP §6, task #10/#15)"},
	{"cannot-use-as:int32", "K1 int locals not pinned to int32 (ROADMAP §6, task #10/#15)"},
	{"cannot-use-as:int", "K1 int locals not pinned to int32 (ROADMAP §6, task #10/#15)"},
	{"mismatched-types:int+float64", "K1 int local mixed with float64 (ROADMAP §6, task #10/#15)"},
	{"mismatched-types:float64+int", "K1 int local mixed with float64 (ROADMAP §6, task #10/#15)"},
	{"mismatched-types:int32+float64", "K1 int32 local mixed with float64 (ROADMAP §6, task #10/#15)"},
	{"mismatched-types:float64+int32", "K1 int32 local mixed with float64 (ROADMAP §6, task #10/#15)"},
	{"truncated:int32", "K1 int32/double mixing (ROADMAP §6, task #10/#15)"},
	// K4: char value/cast — char (rune) vs int promotion not handled, char arith
	// stays rune and prints/uses the code point.
	{"cannot-use-as:rune", "K4 char prints as int code point / rune typing (ROADMAP §6)"},
	{"overflows:rune", "K4 char prints as int code point / rune typing (ROADMAP §6)"},
	{"truncated:rune", "K4 char prints as int code point / rune typing (ROADMAP §6)"},
	{"mismatched-types:rune+int", "K4 char prints as int code point / rune typing (ROADMAP §6)"},
	{"mismatched-types:int+rune", "K4 char prints as int code point / rune typing (ROADMAP §6)"},
	{"cause:K4-char-codepoint", "K4 char value prints code point / replacement char (ROADMAP §6)"},
	// K5: long shift over-masked (5 bits not 6) / long arithmetic value divergence.
	{"cause:long-shift-or-wrap(K5)", "K5 long shift over-masked or long wrap (ROADMAP §6)"},
	// K18: integral double prints without the trailing .0.
	{"cause:K18-double-dot-zero", "K18(new) integral double prints '96' not '96.0' (ROADMAP §2, task #14)"},

	// --- fuzzer-found root causes already catalogued (task #14) ---
	// K12: >>>= compound assign emits an undefined, non-assigning Go call.
	{"undefined: V", "K12 >>>= emits undefined UnsignedRightShiftAssignment (ROADMAP §6/§1, task #14)"},
	// K13: unused local emitted without `_ = v`; valid Java, invalid Go.
	{"declared and not used", "K13 unused local not discarded (ROADMAP §1, task #14)"},

	// --- fuzzer-found root causes NOT yet in KNOWN_ISSUES.md (reported; task #14) ---
	// K16(proposed): -9223372036854775808L emitted as -int64(9223372036854775808).
	{"overflows:int64", "K16(new) negative long-min literal mis-emitted (ROADMAP §6, task #14)"},
	// K17(proposed): explicit upcast (Super) sub emits a failing Go type assertion.
	{"interface conversion", "K17(new) upcast emits failing type assertion (ROADMAP §4/§6, task #14)"},
	// K19(proposed): ternary ?: lowered to stdjava.Ternary(cond,a,b) call, which
	// eagerly evaluates BOTH branches (Java short-circuits). Surfaces as the
	// untaken branch panicking (slice/index out of range, NPE, etc.).
	{"slice V out of range", "K19(new) ternary eagerly evaluates both branches (ROADMAP §6, task #14)"},
	{"index out of range", "K19(new) ternary eagerly evaluates both branches (ROADMAP §6, task #14)"},
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
	// K18(proposed): integral double prints without a decimal point.
	"DoubleFormat": "K18(new) integral double prints '96' not '96.0' (println(double); ROADMAP §2, task #14)",
	// K4-related: string + char concatenates the numeric code, not the glyph.
	"StringPlusChar": "K4 string + char concatenates the code point, not the glyph (ROADMAP §6, task #14)",
}

// IsKnownOpenCorpus reports whether a corpus file (by stem) is a still-open bug
// tracked outside the signature list.
func IsKnownOpenCorpus(stem string) (string, bool) {
	r, ok := knownOpenCorpus[stem]
	return r, ok
}
