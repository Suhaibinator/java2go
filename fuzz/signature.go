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
	case TranspileCrash:
		return string(TranspileCrash) + ":" + normalizeErr(r.Detail)
	case OutputMismatch:
		return string(OutputMismatch) + ":" + mismatchShape(r.JavaOut, r.GoOut)
	}
	return string(r.Category)
}

var goErrLine = regexp.MustCompile(`(?m)^.*\.go:\d+:\d+:\s*(.*)$`)

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
	reIdent   = regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*\d+\b`) // i12, arr3, seed_4
	reNum     = regexp.MustCompile(`-?\b\d+\b`)
	reHex     = regexp.MustCompile(`0x[0-9A-Fa-f]+`)
	reSpaces  = regexp.MustCompile(`\s+`)
	reQuoted  = regexp.MustCompile(`"[^"]*"`)
	reBracket = regexp.MustCompile(`\[[^\]]*\]`)
)

// normalizeErr canonicalizes a compiler/transpiler message so structurally
// identical errors over different programs map to the same string.
func normalizeErr(msg string) string {
	msg = reHex.ReplaceAllString(msg, "H")
	msg = reQuoted.ReplaceAllString(msg, "Q")
	msg = reBracket.ReplaceAllString(msg, "[]")
	msg = reIdent.ReplaceAllString(msg, "V")
	msg = reNum.ReplaceAllString(msg, "N")
	msg = reSpaces.ReplaceAllString(msg, " ")
	return strings.TrimSpace(msg)
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
	case reNum.MatchString(s) && !strings.ContainsAny(s, ".eE"):
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
