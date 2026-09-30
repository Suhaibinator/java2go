// Package fuzz implements a differential fuzzer for java2go: it generates random
// but valid Java programs with deterministic output, runs them on a real JDK and
// on the transpiled-Go side, and reports any behavioral divergence.
//
// The generator (this file) is intentionally restricted to the currently
// supported feature set (see ROADMAP.md ticked items and testfiles/e2e), but it
// is weighted toward the semantic edge cases that have historically been buggy:
// integer overflow, shifts (incl. >>> and counts >= the type width), char
// arithmetic and casts, string+numeric concatenation, division/modulo near zero,
// deeply nested expressions, and ++/-- in expression position.
//
// Every generated program is a single `public class Gen` with a `public static
// void main` that ends by printing computed values, so its stdout is a stable
// oracle. No randomness, time, or threads ever appear in generated code: all
// non-determinism lives in the generator's seeded RNG, not the program.
package fuzz

import (
	"fmt"
	"math/rand"
	"strings"
)

// GenConfig tunes how a single program is generated. The zero value is usable;
// NewGenerator fills in sensible defaults.
type GenConfig struct {
	// MaxStatements bounds the number of top-level statements in main (before the
	// trailing prints). Helpers may add more.
	MaxStatements int
	// MaxExprDepth bounds nesting depth of generated expressions.
	MaxExprDepth int
	// EmitClasses enables generation of an auxiliary user class with methods and
	// (optionally) inheritance, exercising dispatch.
	EmitClasses bool
}

func defaultConfig() GenConfig {
	return GenConfig{
		MaxStatements: 14,
		MaxExprDepth:  4,
		EmitClasses:   true,
	}
}

// jType is a Java primitive/reference type the generator tracks for locals so it
// can keep expressions well-typed.
type jType int

const (
	tInt jType = iota
	tLong
	tDouble
	tBool
	tChar
	tString
)

func (t jType) java() string {
	switch t {
	case tInt:
		return "int"
	case tLong:
		return "long"
	case tDouble:
		return "double"
	case tBool:
		return "boolean"
	case tChar:
		return "char"
	case tString:
		return "String"
	}
	return "int"
}

// numeric reports whether t participates in integer/float arithmetic.
func (t jType) numeric() bool {
	return t == tInt || t == tLong || t == tDouble || t == tChar
}

// integral reports whether t supports bitwise/shift operators.
func (t jType) integral() bool {
	return t == tInt || t == tLong || t == tChar
}

// generator holds the per-program RNG and the set of in-scope locals.
type generator struct {
	rng  *rand.Rand
	cfg  GenConfig
	seed int64

	// locals maps a Java type to the variable names currently declared with that
	// type, so expression generation can reference them.
	locals map[jType][]string
	nextID int

	// prints accumulates expressions whose values are printed at the end of main,
	// forming the program's observable output.
	prints []string
}

// Generate produces a complete, compilable Java program for the given seed. The
// same seed always yields byte-identical source, which is what makes divergences
// reproducible and shrinkable.
func Generate(seed int64) string {
	return GenerateWithConfig(seed, defaultConfig())
}

// GenerateWithConfig is Generate with explicit tuning.
func GenerateWithConfig(seed int64, cfg GenConfig) string {
	g := &generator{
		rng:    rand.New(rand.NewSource(seed)),
		cfg:    cfg,
		seed:   seed,
		locals: map[jType][]string{},
	}
	return g.program()
}

func (g *generator) program() string {
	var b strings.Builder
	fmt.Fprintf(&b, "// seed: %d\n", g.seed)

	var aux string
	if g.cfg.EmitClasses && g.rng.Intn(2) == 0 {
		aux = g.auxClass()
	}

	b.WriteString("public class Gen {\n")
	if aux != "" {
		b.WriteString(aux)
	}
	b.WriteString("    public static void main(String[] args) {\n")

	n := 3 + g.rng.Intn(g.cfg.MaxStatements)
	for i := 0; i < n; i++ {
		stmt := g.statement(2)
		if stmt == "" {
			continue
		}
		for _, line := range strings.Split(strings.TrimRight(stmt, "\n"), "\n") {
			fmt.Fprintf(&b, "        %s\n", line)
		}
	}

	// Always print at least one computed value so stdout is non-empty.
	g.ensurePrints()
	for _, p := range g.prints {
		fmt.Fprintf(&b, "        System.out.println(%s);\n", p)
	}

	b.WriteString("    }\n")
	b.WriteString("}\n")
	return b.String()
}

// auxClass emits a small user class (sometimes with a subclass) exercising
// constructor/field/method dispatch. The class is named Box; if a subclass is
// emitted it is DoubleBox and overrides a method, exercising virtual dispatch.
// Calls to it are appended as prints by the caller via fields recorded here.
func (g *generator) auxClass() string {
	withSub := g.rng.Intn(2) == 0
	var b strings.Builder
	b.WriteString("    static class Box {\n")
	b.WriteString("        int v;\n")
	b.WriteString("        Box(int v) { this.v = v; }\n")
	b.WriteString("        int scale(int k) { return v * k; }\n")
	b.WriteString("        int kind() { return 1; }\n")
	b.WriteString("    }\n")
	if withSub {
		b.WriteString("    static class DoubleBox extends Box {\n")
		b.WriteString("        DoubleBox(int v) { super(v); }\n")
		b.WriteString("        int scale(int k) { return v * k * 2; }\n")
		b.WriteString("        int kind() { return 2; }\n")
		b.WriteString("    }\n")
	}
	// Queue up calls that dispatch through Box / a Box-typed reference to a
	// DoubleBox, so output depends on correct virtual dispatch.
	v := 2 + g.rng.Intn(6)
	k := 1 + g.rng.Intn(5)
	g.prints = append(g.prints, fmt.Sprintf("new Box(%d).scale(%d)", v, k))
	g.prints = append(g.prints, "new Box(3).kind()")
	if withSub {
		g.prints = append(g.prints, fmt.Sprintf("new DoubleBox(%d).scale(%d)", v, k))
		// Box-typed reference to a DoubleBox must dispatch to the override.
		g.prints = append(g.prints, "((Box) new DoubleBox(5)).kind()")
	}
	return b.String()
}

// ensurePrints guarantees the program prints something, even when no local of a
// printable type was declared.
func (g *generator) ensurePrints() {
	// Print every in-scope local of a numeric/bool/string type so the program's
	// final state is fully observable.
	for _, t := range []jType{tInt, tLong, tDouble, tBool, tChar, tString} {
		for _, name := range g.locals[t] {
			if t == tChar {
				// Printing a char in Java prints the glyph; cast to int as well to
				// observe the code point, which is the more bug-prone path.
				g.prints = append(g.prints, name)
				g.prints = append(g.prints, "(int) "+name)
			} else {
				g.prints = append(g.prints, name)
			}
		}
	}
	if len(g.prints) == 0 {
		g.prints = append(g.prints, "42")
	}
}

func (g *generator) freshName(t jType) string {
	prefixes := map[jType]string{
		tInt: "i", tLong: "l", tDouble: "d", tBool: "b", tChar: "c", tString: "s",
	}
	name := fmt.Sprintf("%s%d", prefixes[t], g.nextID)
	g.nextID++
	g.locals[t] = append(g.locals[t], name)
	return name
}
