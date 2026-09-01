// Package fuzz's replay test re-runs every divergence saved under fuzz/corpus/
// through the live transpiler. It mirrors the e2e suite's skip-with-reason
// pattern: a corpus program whose underlying bug is still open is SKIPPED (it is
// expected to keep diverging), while a program whose bug has been FIXED becomes
// an enforced regression test — it must now match the JDK, and any reintroduced
// divergence fails the build.
//
// Lifecycle of a corpus entry:
//  1. The fuzzer finds a divergence, shrinks it, and saves it here.
//  2. While the bug is open, its signature matches fuzz/known.go, so this test
//     skips it (asserting only that it still diverges — if it has silently been
//     fixed, the skip message says so, prompting promotion).
//  3. When the bug is fixed, its signature is removed from fuzz/known.go; this
//     test then requires the program to reproduce the JDK output exactly.
package fuzz

import (
	"io"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// moduleRoot returns the java2go module root. This test lives in <root>/fuzz.
func moduleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// go test runs with the package dir as CWD, i.e. <root>/fuzz.
	return parentDir(wd)
}

func parentDir(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return p
}

func TestCorpusReplay(t *testing.T) {
	root := moduleRoot(t)
	entries, err := LoadCorpus(root)
	if err != nil {
		t.Fatalf("loading corpus: %v", err)
	}
	if len(entries) == 0 {
		t.Skip("no corpus entries yet")
	}

	h, err := NewHarness(root)
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	t.Cleanup(h.Cleanup)

	for _, e := range entries {
		e := e
		name := relName(root, e.Path)
		t.Run(name, func(t *testing.T) {
			// Each corpus program is self-contained; re-run it from source. The seed
			// is irrelevant to replay (the source is fixed), so use a hash-derived
			// pseudo-seed for an isolated scratch dir.
			res := h.Run(replaySeed(e.Source), e.Source)

			if res.Category == JavaError {
				// The recorded program must still be valid Java; if not, the corpus
				// entry is stale (e.g. JDK upgrade) and should be regenerated.
				t.Fatalf("corpus program no longer valid Java:\n%s", res.Detail)
			}

			sig := Signature(res)
			reason, known := IsKnown(sig)
			// Also consult the per-file skip list for open bugs whose signature
			// shape is too generic to list globally.
			if r, ok := IsKnownOpenCorpus(stem(e.Path)); ok {
				known, reason = true, r
			}

			switch {
			case res.Category == OK:
				// The divergence no longer reproduces — the bug has been fixed. This
				// is the desired end state; matching the JDK passes. (Recorded as a
				// log line so a fixed-but-still-listed marker in known.go is visible.)
				t.Logf("FIXED: %s now matches the JDK", name)
			case known:
				// Still diverging on a known-open bug: skip, exactly like the e2e
				// suite skips a fixture blocked on an open ROADMAP item.
				t.Skipf("known open bug (%s): %s", res.Category, reason)
			default:
				// Diverges, and the signature is NOT marked known-open. This is a
				// regression: a previously-fixed (or never-acknowledged) divergence is
				// live. Fail loudly with the repro.
				t.Fatalf("unexpected divergence [%s] sig=%q\n--- program ---\n%s\n--- expected (java) ---\n%s\n--- got ---\n%s\n%s",
					res.Category, sig, e.Source, res.JavaOut, res.GoOut, res.Detail)
			}
		})
	}
}

// relName trims the module root prefix from a corpus path for a readable subtest
// name.
func relName(root, path string) string {
	if len(path) > len(root)+1 && path[:len(root)] == root {
		return path[len(root)+1:]
	}
	return path
}

// stem returns the file basename without directory or extension, e.g.
// ".../DoubleFormat.java" -> "DoubleFormat".
func stem(path string) string {
	base := path
	if i := lastSlash(path); i >= 0 {
		base = path[i+1:]
	}
	if i := lastDot(base); i >= 0 {
		base = base[:i]
	}
	return base
}

func lastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

func lastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}

// replaySeed derives a stable per-program scratch id from its content hash so
// concurrent replay subtests use distinct work dirs.
func replaySeed(source string) int64 {
	var h int64 = 1469598103934665603
	for i := 0; i < len(source); i++ {
		h ^= int64(source[i])
		h *= 1099511628211
	}
	if h < 0 {
		h = -h
	}
	return h
}
