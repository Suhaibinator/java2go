// Command javafuzz is the differential fuzzer driver for java2go. It generates
// random valid Java programs, runs them on the JDK and on the transpiled Go, and
// reports behavioral divergences, saving each unique one to fuzz/corpus/.
//
// Usage:
//
//	javafuzz -n 500 -seed 1 -root .            # run 500 seeds starting at 1
//	javafuzz -seed 12345 -only                 # run a single seed, print its program
//
// The seed range is deterministic: seed s always produces the same program, so a
// reported divergence is reproducible with `javafuzz -seed s -only`.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	fuzz "github.com/NickyBoy89/java2go/fuzz"
	log "github.com/sirupsen/logrus"
)

func main() {
	var (
		root     = flag.String("root", ".", "java2go module root (contains go.mod)")
		start    = flag.Int64("seed", 1, "first seed (or the only seed with -only)")
		count    = flag.Int("n", 200, "number of seeds to run")
		only     = flag.Bool("only", false, "run just -seed and print its generated program + outcome")
		noShrink = flag.Bool("no-shrink", false, "skip shrinking divergences (faster triage)")
		quiet    = flag.Bool("quiet", false, "suppress per-divergence detail, print only the summary")
	)
	flag.Parse()

	// The transpiler is chatty on the global logger; silence it.
	log.SetOutput(io.Discard)

	h, err := fuzz.NewHarness(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "harness:", err)
		os.Exit(2)
	}
	defer h.Cleanup()
	moduleRoot := *root

	if *only {
		runSingle(h, moduleRoot, *start)
		return
	}

	runBatch(h, moduleRoot, *start, *count, !*noShrink, *quiet)
}

// runSingle prints one program and its differential outcome — the reproduction
// entry point for a reported seed.
func runSingle(h *fuzz.Harness, root string, seed int64) {
	src := fuzz.Generate(seed)
	fmt.Println(src)
	res := h.Run(seed, src)
	fmt.Println("==== category:", res.Category)
	if res.Category == fuzz.OutputMismatch {
		fmt.Println("---- java ----")
		fmt.Println(res.JavaOut)
		fmt.Println("---- go ----")
		fmt.Println(res.GoOut)
	} else if res.Detail != "" {
		fmt.Println("---- detail ----")
		fmt.Println(res.Detail)
	}
}

// bug bundles a saved corpus entry with its root-cause signature and known-bug
// status, so the report can separate genuinely-new divergences from already
// tracked ones.
type bug struct {
	entry  fuzz.CorpusEntry
	sig    string
	known  bool
	reason string
}

// runBatch is the main fuzzing loop. It classifies each seed, shrinks divergences
// to a minimal repro, dedups by ROOT-CAUSE SIGNATURE (not by program), saves one
// corpus entry per unique signature, and prints a summary that surfaces NEW bugs
// separately from already-known ones.
func runBatch(h *fuzz.Harness, root string, start int64, count int, shrink, quiet bool) {
	counts := map[fuzz.Category]int{}
	seenSig := map[string]bool{}
	var bugs []bug

	for i := 0; i < count; i++ {
		seed := start + int64(i)
		src := fuzz.Generate(seed)
		res := h.Run(seed, src)
		counts[res.Category]++

		if !res.Diverged() {
			continue
		}

		// Dedup by signature BEFORE the expensive shrink, so each root cause is
		// shrunk and saved exactly once.
		sig := fuzz.Signature(res)
		if seenSig[sig] {
			continue
		}
		seenSig[sig] = true

		final := res
		if shrink {
			shrunk := fuzz.Shrink(h, seed, res.Source, res.Category)
			rerun := h.Run(seed, shrunk)
			// Keep the shrink only if it preserved the same root-cause signature;
			// otherwise it reduced to a different bug, so report the original.
			if rerun.Category == res.Category && fuzz.Signature(rerun) == sig {
				final = rerun
			}
		}

		entry := fuzz.CorpusEntry{
			Category: final.Category,
			Seed:     seed,
			Source:   final.Source,
			Expected: final.JavaOut,
			Actual:   corpusActual(final),
		}
		if path, err := entry.Save(root); err != nil {
			fmt.Fprintln(os.Stderr, "saving corpus entry:", err)
		} else {
			entry.Path = path
		}

		reason, known := fuzz.IsKnown(sig)
		b := bug{entry: entry, sig: sig, known: known, reason: reason}
		bugs = append(bugs, b)

		if !quiet && !known {
			printDivergence(seed, entry)
		}
	}

	printSummary(count, counts, bugs)
}

// corpusActual picks the most useful "actual" text for the corpus: the Go stdout
// for a mismatch, or the error detail for a crash/compile failure.
func corpusActual(r fuzz.Result) string {
	if r.Category == fuzz.OutputMismatch {
		return r.GoOut
	}
	return r.Detail
}

func printDivergence(seed int64, e fuzz.CorpusEntry) {
	fmt.Printf("\n===== %s  seed=%d  =====\n", e.Category, seed)
	fmt.Println(e.Source)
	if e.Category == fuzz.OutputMismatch {
		fmt.Println("--- expected (java) ---")
		fmt.Println(e.Expected)
		fmt.Println("--- actual (go) ---")
		fmt.Println(e.Actual)
	} else {
		fmt.Println("--- detail ---")
		fmt.Println(e.Actual)
	}
	if e.Path != "" {
		fmt.Println("saved:", e.Path)
	}
}

func printSummary(total int, counts map[fuzz.Category]int, bugs []bug) {
	fmt.Printf("\n========== SUMMARY ==========\n")
	fmt.Printf("programs run: %d\n", total)
	order := []fuzz.Category{fuzz.OK, fuzz.JavaError, fuzz.TranspileCrash, fuzz.GoCompileError, fuzz.OutputMismatch}
	for _, c := range order {
		fmt.Printf("  %-16s %d\n", c, counts[c])
	}

	var newBugs, knownBugs []bug
	for _, b := range bugs {
		if b.known {
			knownBugs = append(knownBugs, b)
		} else {
			newBugs = append(newBugs, b)
		}
	}
	sort.Slice(newBugs, func(i, j int) bool { return newBugs[i].entry.Seed < newBugs[j].entry.Seed })
	sort.Slice(knownBugs, func(i, j int) bool { return knownBugs[i].entry.Seed < knownBugs[j].entry.Seed })

	fmt.Printf("unique root causes: %d new, %d known-and-tracked\n", len(newBugs), len(knownBugs))

	if len(newBugs) > 0 {
		fmt.Printf("\n--- NEW unique root causes ---\n")
		for _, b := range newBugs {
			fmt.Printf("  [%s] seed=%d  sig=%q\n      %s\n", b.entry.Category, b.entry.Seed, b.sig, b.entry.Path)
		}
	}
	if len(knownBugs) > 0 {
		fmt.Printf("\n--- known/tracked root causes (saved as regression inputs) ---\n")
		for _, b := range knownBugs {
			fmt.Printf("  [%s] seed=%d  %s\n      sig=%q\n", b.entry.Category, b.entry.Seed, b.reason, b.sig)
		}
	}
}
