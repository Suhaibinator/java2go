package fuzz

import (
	"strings"
)

// Shrink minimizes a divergent program while preserving the SAME divergence
// category. It works line-by-line (the generator emits one logical unit per
// line, so this is a robust delta-debugging granularity): it repeatedly tries to
// drop body statements, keeping a drop only if the reduced program still
// reproduces the original category. Declarations referenced by surviving lines
// are kept implicitly because dropping them flips the program to JAVA_ERROR or
// GO_COMPILE_ERROR (a different category) and is rejected.
//
// The class/main scaffolding and the trailing println block are preserved so the
// result stays a runnable program. Shrinking is best-effort: if no reduction
// reproduces, the original source is returned unchanged.
func Shrink(h *Harness, seed int64, source string, target Category) string {
	targetSig := Signature(h.Run(seed, source))
	best := source

	// Bound total reduction work so a pathological program cannot stall the
	// fuzzer; each pass is O(lines) harness runs and passes converge quickly.
	const maxPasses = 40

	// Iterate to a fixpoint: each pass may enable further drops.
	for pass := 0; pass < maxPasses; pass++ {
		lines := strings.Split(best, "\n")
		reduced := false

		for i := range lines {
			if !isReducibleLine(lines[i]) {
				continue
			}
			candidate := dropLine(lines, i)
			if reproduces(h, seed, candidate, target, targetSig) {
				best = candidate
				reduced = true
				break
			}
		}

		if !reduced {
			break
		}
	}

	// Second phase: try dropping trailing println lines (each is independent), so
	// a mismatch caused by one printed value shrinks to just that value.
	best = shrinkPrints(h, seed, best, target, targetSig)
	return best
}

// reproduces reports whether candidate still yields the SAME root-cause
// divergence: same category and same normalized signature. Requiring the
// signature to match prevents the shrinker from collapsing one bug onto a
// different one that happens to share a category. A candidate that no longer
// compiles as Java does not count.
func reproduces(h *Harness, seed int64, candidate string, target Category, targetSig string) bool {
	res := h.Run(seed, candidate)
	return res.Category == target && Signature(res) == targetSig
}

// isReducibleLine reports whether a line is a candidate for removal: a statement
// inside main, not structural scaffolding (class header, main header, braces) and
// not a println (handled separately so we can shrink the print block last).
func isReducibleLine(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" {
		return false
	}
	if strings.HasPrefix(t, "//") {
		return false
	}
	if strings.HasPrefix(t, "public class") ||
		strings.HasPrefix(t, "public static void main") ||
		strings.HasPrefix(t, "static class") ||
		strings.HasPrefix(t, "import ") ||
		t == "}" || t == "{" || strings.HasPrefix(t, "} else") {
		return false
	}
	if strings.Contains(t, "System.out.println") {
		return false
	}
	// Avoid dropping a lone control-flow header, which would orphan its block.
	if strings.HasSuffix(t, "{") {
		return false
	}
	return true
}

// dropLine returns source (as a line slice) with index i removed, rejoined.
func dropLine(lines []string, i int) string {
	out := make([]string, 0, len(lines)-1)
	out = append(out, lines[:i]...)
	out = append(out, lines[i+1:]...)
	return strings.Join(out, "\n")
}

// shrinkPrints removes trailing println statements one at a time while the
// divergence persists, so an OUTPUT_MISMATCH collapses to the minimal set of
// printed values that still differ.
func shrinkPrints(h *Harness, seed int64, source string, target Category, targetSig string) string {
	best := source
	for {
		lines := strings.Split(best, "\n")
		reduced := false
		for i := range lines {
			if !strings.Contains(lines[i], "System.out.println") {
				continue
			}
			candidate := dropLine(lines, i)
			if reproduces(h, seed, candidate, target, targetSig) {
				best = candidate
				reduced = true
				break
			}
		}
		if !reduced {
			break
		}
	}
	return best
}
