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

// mismatchShape summarizes an output mismatch by the index and type of the first
// differing line, so e.g. "the 3rd printed int differs" forms a key. It does not
// encode the exact values (those vary per program), only the structural locus.
func mismatchShape(java, goOut string) string {
	jl := strings.Split(normalize(java), "\n")
	gl := strings.Split(normalize(goOut), "\n")
	n := len(jl)
	if len(gl) < n {
		n = len(gl)
	}
	for i := 0; i < n; i++ {
		if jl[i] != gl[i] {
			return "line" + itoa(i) + ":" + valueShape(jl[i]) + "vs" + valueShape(gl[i])
		}
	}
	if len(jl) != len(gl) {
		return "linecount"
	}
	return "unknown"
}

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
