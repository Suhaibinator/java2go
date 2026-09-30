package fuzz

import (
	"fmt"
	"math/rand"
)

// atom returns a leaf expression of type t: an in-scope local (preferred when
// available) or a literal weighted toward boundary values.
func (g *generator) atom(t jType) string {
	if locals := g.locals[t]; len(locals) > 0 && g.rng.Intn(2) == 0 {
		return locals[g.rng.Intn(len(locals))]
	}
	switch t {
	case tInt:
		return g.intAtom()
	case tLong:
		return g.longAtom()
	case tDouble:
		return g.doubleAtom()
	case tBool:
		return g.boolAtom()
	case tChar:
		return g.charAtom()
	case tString:
		return fmt.Sprintf("%q", g.word())
	}
	return "0"
}

// intBoundaries are the integer literals most likely to surface overflow,
// sign-extension, and shift bugs.
var intBoundaries = []string{
	"0", "1", "-1", "2", "-2", "7", "-7", "255", "256", "65535", "65536",
	"2147483647", "-2147483648", "1000000", "-1000000", "0x7FFFFFFF", "0xFFFFFFFF",
	"100", "-100", "3", "10",
}

func (g *generator) intAtom() string {
	if v, ok := g.maybeIntLocal(); ok {
		return v
	}
	if g.rng.Intn(3) == 0 {
		// Small random value to add variety beyond fixed boundaries.
		return fmt.Sprintf("%d", g.rng.Intn(2000)-1000)
	}
	return intBoundaries[g.rng.Intn(len(intBoundaries))]
}

func (g *generator) maybeIntLocal() (string, bool) {
	locals := g.locals[tInt]
	if len(locals) > 0 && g.rng.Intn(2) == 0 {
		return locals[g.rng.Intn(len(locals))], true
	}
	return "", false
}

// nonZeroIntAtom returns a divisor guaranteed non-zero at runtime. A bare local
// could hold 0, so when a variable is chosen it is OR'd with 1 (forcing it odd,
// hence non-zero) — this keeps divisor-near-zero coverage without ever throwing
// ArithmeticException, which would be a generator bug, not a transpiler bug.
func (g *generator) nonZeroIntAtom() string {
	if locals := g.locals[tInt]; len(locals) > 0 && g.rng.Intn(2) == 0 {
		return "(" + locals[g.rng.Intn(len(locals))] + " | 1)"
	}
	for i := 0; i < 8; i++ {
		a := intBoundaries[g.rng.Intn(len(intBoundaries))]
		if a != "0" {
			return a
		}
	}
	return "3"
}

var longBoundaries = []string{
	"0L", "1L", "-1L", "10000000000L", "-10000000000L",
	"9223372036854775807L", "-9223372036854775808L", "4294967296L",
	"1000000000000L", "2147483648L", "100L",
}

func (g *generator) longAtom() string {
	locals := g.locals[tLong]
	if len(locals) > 0 && g.rng.Intn(2) == 0 {
		return locals[g.rng.Intn(len(locals))]
	}
	return longBoundaries[g.rng.Intn(len(longBoundaries))]
}

func (g *generator) nonZeroLongAtom() string {
	if locals := g.locals[tLong]; len(locals) > 0 && g.rng.Intn(2) == 0 {
		return "(" + locals[g.rng.Intn(len(locals))] + " | 1L)"
	}
	for i := 0; i < 8; i++ {
		a := longBoundaries[g.rng.Intn(len(longBoundaries))]
		if a != "0L" {
			return a
		}
	}
	return "3L"
}

var doubleBoundaries = []string{
	"0.0", "1.0", "-1.0", "0.5", "3.14", "-2.5", "100.0", "0.1", "2.0", "10.0",
}

func (g *generator) doubleAtom() string {
	locals := g.locals[tDouble]
	if len(locals) > 0 && g.rng.Intn(2) == 0 {
		return locals[g.rng.Intn(len(locals))]
	}
	return doubleBoundaries[g.rng.Intn(len(doubleBoundaries))]
}

func (g *generator) nonZeroDoubleAtom() string {
	for i := 0; i < 8; i++ {
		a := g.doubleAtom()
		if a != "0.0" {
			return a
		}
	}
	return "2.0"
}

func (g *generator) boolAtom() string {
	locals := g.locals[tBool]
	if len(locals) > 0 && g.rng.Intn(2) == 0 {
		return locals[g.rng.Intn(len(locals))]
	}
	if g.rng.Intn(2) == 0 {
		return "true"
	}
	return "false"
}

var charBoundaries = []string{
	"'A'", "'a'", "'Z'", "'z'", "'0'", "'9'", "' '", "'~'", "'!'",
}

func (g *generator) charAtom() string {
	locals := g.locals[tChar]
	if len(locals) > 0 && g.rng.Intn(2) == 0 {
		return locals[g.rng.Intn(len(locals))]
	}
	return charBoundaries[g.rng.Intn(len(charBoundaries))]
}

// shiftAmount returns an int shift count, deliberately including values >= 32 to
// exercise Java's 5-bit masking for int shifts.
func (g *generator) shiftAmount() string {
	counts := []string{"0", "1", "4", "8", "16", "31", "32", "33", "40", "64"}
	return counts[g.rng.Intn(len(counts))]
}

// longShiftAmount returns counts that probe the 6-bit mask for long shifts.
func (g *generator) longShiftAmount() string {
	counts := []string{"0", "1", "16", "32", "33", "63", "64", "65", "100"}
	return counts[g.rng.Intn(len(counts))]
}

var words = []string{
	"alpha", "beta", "gamma", "x", "n=", "val", "sum=", "k", "-", "::", "out",
}

func (g *generator) word() string {
	return words[g.rng.Intn(len(words))]
}

// weighted picks an index in [0,len(weights)) proportional to its weight.
func (g *generator) weighted(weights []int) int {
	total := 0
	for _, w := range weights {
		total += w
	}
	r := g.rng.Intn(total)
	for i, w := range weights {
		if r < w {
			return i
		}
		r -= w
	}
	return len(weights) - 1
}

func pick(rng *rand.Rand, opts ...string) string {
	return opts[rng.Intn(len(opts))]
}

// pickLocal returns a random in-scope local of any type.
func (g *generator) pickLocal() (jType, string, bool) {
	var all []struct {
		t jType
		n string
	}
	for t, names := range g.locals {
		for _, n := range names {
			all = append(all, struct {
				t jType
				n string
			}{t, n})
		}
	}
	if len(all) == 0 {
		return tInt, "", false
	}
	c := all[g.rng.Intn(len(all))]
	return c.t, c.n, true
}

func (g *generator) pickNumericLocal() (jType, string, bool) {
	var cands []struct {
		t jType
		n string
	}
	for t, names := range g.locals {
		if !t.numeric() && t != tString {
			continue
		}
		for _, n := range names {
			cands = append(cands, struct {
				t jType
				n string
			}{t, n})
		}
	}
	if len(cands) == 0 {
		return tInt, "", false
	}
	c := cands[g.rng.Intn(len(cands))]
	return c.t, c.n, true
}

func (g *generator) pickIntLocal() (jType, string, bool) {
	names := g.locals[tInt]
	if len(names) == 0 {
		return tInt, "", false
	}
	return tInt, names[g.rng.Intn(len(names))], true
}
