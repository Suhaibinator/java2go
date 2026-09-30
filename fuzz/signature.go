package fuzz

import (
	"regexp"
	"strings"
)

// Signature reduces a divergence to a stable, program-independent root-cause key
// so that many seeds exhibiting the same bug collapse to one report. This is the
// dedup axis the task asks for: "dedupe by root cause, not by program."
//
// The signature is derived from the failure category and its detail/diff:
//   - GO_COMPILE_ERROR: the first Go compiler error message, with identifiers,
//     literals, and source positions blanked out so two programs that trip the
//     same typing rule share a key (e.g. every "mismatched types int and int32").
//   - TRANSPILE_CRASH: the panic/error message, similarly normalized.
//   - OUTPUT_MISMATCH: the kind of value that differs (first differing line's
//     shape) — this is coarser and may merge distinct numeric bugs, so callers
//     still shrink+inspect mismatches individually.
func Signature(r Result) string {
	switch r.Category {
	case GoCompileError:
		return string(GoCompileError) + ":" + normalizeErr(firstGoError(r.Detail))
	case GoRuntimeError:
		return string(GoRuntimeError) + ":" + normalizeErr(firstPanic(r.Detail))
	case TranspileCrash:
		return string(TranspileCrash) + ":" + normalizeErr(r.Detail)
	case OutputMismatch:
		return string(OutputMismatch) + ":" + mismatchShape(r.JavaOut, r.GoOut)
	}
	return string(r.Category)
}

var (
	goErrLine = regexp.MustCompile(`(?m)^.*\.go:\d+:\d+:\s*(.*)$`)
	panicLine = regexp.MustCompile(`(?m)^panic:\s*(.*)$`)
)

// firstPanic extracts the Go panic message (the line after "panic:") so distinct
// crash causes form distinct signatures.
func firstPanic(detail string) string {
	if m := panicLine.FindStringSubmatch(detail); m != nil {
		return m[1]
	}
	for _, line := range strings.Split(detail, "\n") {
		if s := strings.TrimSpace(line); s != "" && !strings.HasPrefix(s, "exit status") {
			return s
		}
	}
	return detail
}

// firstGoError pulls the first "file.go:line:col: message" out of go-build output.
func firstGoError(detail string) string {
	m := goErrLine.FindStringSubmatch(detail)
	if m != nil {
		return m[1]
	}
	// Fall back to the first non-empty line.
	for _, line := range strings.Split(detail, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			return s
		}
	}
	return detail
}

var (
	reSpaces = regexp.MustCompile(`\s+`)
	reQuoted = regexp.MustCompile(`"[^"]*"`)

	// goTypes are the Go type names worth PRESERVING in a signature: which types
	// are involved is the discriminating fact of a typing bug, so two
	// "overflows int32" errors must share a key while "overflows rune" stays
	// distinct. Order matters: longer names first so int64 isn't truncated to int.
	goTypes = []string{"int64", "int32", "int16", "int8", "uint64", "uint32",
		"float64", "float32", "rune", "byte", "bool", "string", "int", "uint"}

	// errKinds collapses a compiler error to its KIND, discarding the specific
	// (program-dependent) expression that triggered it. Each pattern keeps only
	// the structural part plus any Go type names it mentions.
	overflowKind  = regexp.MustCompile(`overflows (` + typeAlt() + `)`)
	mismatchKind  = regexp.MustCompile(`mismatched types (` + typeAlt() + `) and (` + typeAlt() + `)`)
	cannotUseKind = regexp.MustCompile(`cannot use .* as (` + typeAlt() + `) value`)
	truncKind     = regexp.MustCompile(`truncated to (` + typeAlt() + `)`)
)

func typeAlt() string { return strings.Join(goTypes, "|") }

// normalizeErr canonicalizes a compiler/transpiler message so structurally
// identical errors over different programs map to the same key. It first tries
// to recognize a known error KIND and reduce to "<kind>:<types>", discarding the
// program-specific expression; otherwise it falls back to blanking identifiers
// and literals while preserving Go type names.
func normalizeErr(msg string) string {
	if m := mismatchKind.FindStringSubmatch(msg); m != nil {
		return "mismatched-types:" + m[1] + "+" + m[2]
	}
	if m := overflowKind.FindStringSubmatch(msg); m != nil {
		return "overflows:" + m[1]
	}
	if m := truncKind.FindStringSubmatch(msg); m != nil {
		return "truncated:" + m[1]
	}
	if m := cannotUseKind.FindStringSubmatch(msg); m != nil {
		return "cannot-use-as:" + m[1]
	}
	// Generic fallback: blank quoted strings, hex, decimals, brackets, and
	// identifiers, but keep Go type keywords.
	msg = reQuoted.ReplaceAllString(msg, "Q")
	msg = regexp.MustCompile(`0x[0-9A-Fa-f]+`).ReplaceAllString(msg, "H")
	msg = regexp.MustCompile(`\[[^\]]*\]`).ReplaceAllString(msg, "[]")
	msg = blankIdents(msg)
	msg = regexp.MustCompile(`-?\b\d+\b`).ReplaceAllString(msg, "N")
	msg = reSpaces.ReplaceAllString(msg, " ")
	return strings.TrimSpace(msg)
}

// goTypeSet is the lookup form of goTypes, for blankIdents.
var goTypeSet = func() map[string]bool {
	m := map[string]bool{}
	for _, t := range goTypes {
		m[t] = true
	}
	return m
}()

var reWord = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

// blankIdents replaces identifiers with "V" but preserves Go type keywords and
// a few structural words, so signatures stay meaningful without encoding the
// program's variable names.
func blankIdents(msg string) string {
	return reWord.ReplaceAllStringFunc(msg, func(w string) string {
		if goTypeSet[w] {
			return w
		}
		switch w {
		case "invalid", "operation", "cannot", "use", "as", "value", "in",
			"assignment", "array", "or", "slice", "literal", "undefined",
			"declared", "and", "not", "used", "constant", "type", "of",
			"len", "take", "address", "stdjava",
			// runtime panic vocabulary
			"interface", "conversion", "index", "out", "range", "nil",
			"pointer", "dereference", "divide", "zero", "runtime", "error":
			return w
		}
		return "V"
	})
}

// mismatchShape summarizes an output mismatch by ROOT CAUSE where recognizable,
// otherwise by the positional shape of the first differing line. The cause
// classifiers are position-independent so the same value-level bug (e.g. a char
// printed as its code point) dedups across programs regardless of where it lands.
func mismatchShape(java, goOut string) string {
	jl := strings.Split(normalize(java), "\n")
	gl := strings.Split(normalize(goOut), "\n")
	n := len(jl)
	if len(gl) < n {
		n = len(gl)
	}
	for i := 0; i < n; i++ {
		if jl[i] != gl[i] {
			if c := classifyDiff(jl[i], gl[i]); c != "" {
				return c
			}
			return "line" + itoa(i) + ":" + valueShape(jl[i]) + "vs" + valueShape(gl[i])
		}
	}
	if len(jl) != len(gl) {
		return "linecount"
	}
	return "unknown"
}

var (
	reReplacementChar = regexp.MustCompile("�")
	reAllDigits       = regexp.MustCompile(`^-?\d+$`)
	reFloatDotZero    = regexp.MustCompile(`^-?\d+\.0+$`)
)

// classifyDiff recognizes the signature value-level divergences so they bucket to
// their KNOWN_ISSUES id regardless of program/position. Returns "" if unrecognized.
func classifyDiff(j, g string) string {
	js, gs := strings.TrimSpace(j), strings.TrimSpace(g)

	// K4: char rendered as a glyph in Java but as a replacement char or its
	// numeric code point in Go (or vice-versa). Detect a Unicode replacement char
	// on either side, or a non-numeric Java token vs a (large/negative) numeric Go
	// token of the same surrounding text.
	if reReplacementChar.MatchString(g) || reReplacementChar.MatchString(j) {
		return "cause:K4-char-codepoint"
	}

	// K18: Java prints an integral double with a trailing ".0", Go prints it as a
	// bare integer (e.g. "96.0" vs "96").
	if reFloatDotZero.MatchString(js) && reAllDigits.MatchString(gs) {
		return "cause:K18-double-dot-zero"
	}

	// K5/long: both sides are integers but differ in magnitude with at least one
	// exceeding the 32-bit range — characteristic of long-shift over-masking or
	// long-arithmetic wrap.
	if reAllDigits.MatchString(js) && reAllDigits.MatchString(gs) && js != gs {
		if exceeds32(js) || exceeds32(gs) {
			return "cause:long-shift-or-wrap(K5)"
		}
	}
	return ""
}

// exceeds32 reports whether a decimal integer string is outside the signed
// 32-bit range, a heuristic for "this is long arithmetic".
func exceeds32(s string) bool {
	n, err := parseInt64(s)
	if err != nil {
		return false
	}
	return n > 2147483647 || n < -2147483648
}

func parseInt64(s string) (int64, error) {
	var n int64
	neg := false
	i := 0
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		neg = s[i] == '-'
		i++
	}
	if i == len(s) {
		return 0, errEmpty
	}
	for ; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, errEmpty
		}
		n = n*10 + int64(s[i]-'0')
	}
	if neg {
		n = -n
	}
	return n, nil
}

var errEmpty = fmtError("not an int")

type fmtError string

func (e fmtError) Error() string { return string(e) }

// valueShape classifies a printed token as int/float/bool/word so two mismatches
// of the same kind cluster.
func valueShape(s string) string {
	s = strings.TrimSpace(s)
	switch {
	case s == "true" || s == "false":
		return "bool"
	case regexp.MustCompile(`^-?\d+$`).MatchString(s):
		return "int"
	case strings.ContainsAny(s, ".") && regexp.MustCompile(`^-?\d`).MatchString(s):
		return "float"
	default:
		return "word"
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [20]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		p--
		b[p] = '-'
	}
	return string(b[p:])
}
