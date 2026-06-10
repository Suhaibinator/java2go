package fuzz

import (
	"fmt"
	"strings"
)

// statement generates a single top-level statement (possibly multi-line). budget
// limits recursion into nested blocks.
func (g *generator) statement(budget int) string {
	// Weighted choice over statement kinds. Declarations dominate early so later
	// statements have locals to work with.
	switch g.weighted([]int{28, 13, 10, 9, 8, 8, 8, 6, 6, 6}) {
	case 0:
		return g.declStmt()
	case 1:
		return g.assignStmt()
	case 2:
		return g.compoundAssignStmt()
	case 3:
		return g.incDecStmt()
	case 4:
		if budget <= 0 {
			return g.declStmt()
		}
		return g.ifStmt(budget)
	case 5:
		if budget <= 0 {
			return g.declStmt()
		}
		return g.forStmt(budget)
	case 6:
		if budget <= 0 {
			return g.declStmt()
		}
		return g.whileStmt(budget)
	case 7:
		return g.stringBuilderStmt()
	case 8:
		return g.arrayStmt()
	case 9:
		return g.stringMethodStmt()
	}
	return g.declStmt()
}

// nestedStatement generates a statement for a block body. It deliberately avoids
// introducing new locals: a local declared inside a block is scope-restored when
// the block closes, so it would never be printed and would surface in Go as a
// "declared and not used" error (Java tolerates unused locals; Go does not). To
// keep generated programs valid Go without masking real bugs, block bodies only
// mutate already-in-scope locals or nest further control flow.
func (g *generator) nestedStatement(budget int) string {
	// Without an in-scope numeric local to mutate, the only no-new-local options
	// are nested control flow (which also needs a body); fall back to an empty
	// body marker the block emitters turn into "{}".
	_, _, hasNum := g.pickNumericLocal()
	if !hasNum {
		if budget > 0 && g.rng.Intn(2) == 0 {
			return g.forStmt(budget)
		}
		return ""
	}
	switch g.weighted([]int{34, 24, 18, 12, 12}) {
	case 0:
		return g.assignStmt()
	case 1:
		return g.compoundAssignStmt()
	case 2:
		return g.incDecStmt()
	case 3:
		if budget <= 0 {
			return g.compoundAssignStmt()
		}
		return g.ifStmt(budget)
	case 4:
		if budget <= 0 {
			return g.compoundAssignStmt()
		}
		return g.forStmt(budget)
	}
	return g.compoundAssignStmt()
}

// declStmt declares a new local initialized to a random expression of a chosen
// type, biased toward numeric/edge-case types.
func (g *generator) declStmt() string {
	t := g.pickDeclType()
	// Generate the initializer BEFORE the variable enters scope, so it can never
	// reference itself (which Java rejects as "variable might not be initialized").
	init := g.expr(t, g.cfg.MaxExprDepth)
	name := g.freshName(t)
	return fmt.Sprintf("%s %s = %s;", t.java(), name, init)
}

func (g *generator) pickDeclType() jType {
	// Weight int/long/char heavily — those carry the overflow/shift/cast bugs.
	types := []jType{tInt, tLong, tDouble, tChar, tBool, tString}
	idx := g.weighted([]int{30, 18, 14, 14, 12, 12})
	return types[idx]
}

// assignStmt reassigns an existing local (same type) to a new expression.
func (g *generator) assignStmt() string {
	t, name, ok := g.pickLocal()
	if !ok {
		return g.declStmt()
	}
	return fmt.Sprintf("%s = %s;", name, g.expr(t, g.cfg.MaxExprDepth))
}

// compoundAssignStmt emits x OP= y for a numeric/integral local, hitting the
// in-place arithmetic path (overflow, shift-assign, etc.).
func (g *generator) compoundAssignStmt() string {
	t, name, ok := g.pickNumericLocal()
	if !ok {
		return g.declStmt()
	}
	var ops []string
	switch {
	case t == tString:
		ops = []string{"+="}
	case t.integral():
		ops = []string{"+=", "-=", "*=", "&=", "|=", "^=", "<<=", ">>=", ">>>="}
	default:
		ops = []string{"+=", "-=", "*="}
	}
	op := ops[g.rng.Intn(len(ops))]
	rhsType := t
	if strings.HasPrefix(op, "<<") || strings.HasPrefix(op, ">>") {
		rhsType = tInt
		return fmt.Sprintf("%s %s %s;", name, op, g.shiftAmount())
	}
	return fmt.Sprintf("%s %s %s;", name, op, g.expr(rhsType, 1))
}

// incDecStmt emits a standalone ++/-- on a numeric local.
func (g *generator) incDecStmt() string {
	t, name, ok := g.pickNumericLocal()
	if !ok || t == tString || t == tDouble {
		return g.compoundAssignStmt()
	}
	if g.rng.Intn(2) == 0 {
		return name + "++;"
	}
	return name + "--;"
}

// ifStmt emits an if (optionally with else), each branch a nested statement.
// Nested blocks run under a saved scope so locals declared inside them do not
// leak into the outer print list (they would be out of scope there).
func (g *generator) ifStmt(budget int) string {
	cond := g.expr(tBool, 2)
	var b strings.Builder
	fmt.Fprintf(&b, "if (%s) {\n", cond)
	b.WriteString(indent(g.scoped(func() string { return g.nestedStatement(budget - 1) })))
	if g.rng.Intn(2) == 0 {
		b.WriteString("} else {\n")
		b.WriteString(indent(g.scoped(func() string { return g.nestedStatement(budget - 1) })))
	}
	b.WriteString("}")
	return b.String()
}

// scoped runs fn with the current local scope saved, restoring it afterward so
// declarations made inside a nested block are forgotten once the block closes.
func (g *generator) scoped(fn func() string) string {
	saved := make(map[jType][]string, len(g.locals))
	for t, names := range g.locals {
		saved[t] = append([]string(nil), names...)
	}
	out := fn()
	g.locals = saved
	return out
}

// forStmt emits a counted for loop that accumulates into an existing int local
// (or a fresh one), exercising loop-var typing and in-loop arithmetic.
func (g *generator) forStmt(budget int) string {
	var prefix string
	_, accName, ok := g.pickIntLocal()
	if !ok {
		accName = g.freshName(tInt)
		prefix = fmt.Sprintf("int %s = 0;\n", accName)
	}
	iv := fmt.Sprintf("it%d", g.nextID)
	g.nextID++
	lo := g.rng.Intn(3)
	hi := lo + 1 + g.rng.Intn(5)
	body := g.scoped(func() string { return g.loopBody(accName, budget) })
	return fmt.Sprintf("%sfor (int %s = %d; %s < %d; %s++) {\n%s}",
		prefix, iv, lo, iv, hi, iv, indent(body))
}

// whileStmt emits a while loop with a counter that is guaranteed to terminate.
func (g *generator) whileStmt(budget int) string {
	limit := 1 + g.rng.Intn(5)
	_, accName, ok := g.pickIntLocal()
	cnt := g.freshName(tInt)
	var b strings.Builder
	if !ok {
		accName = g.freshName(tInt)
		fmt.Fprintf(&b, "int %s = 0;\n", accName)
	}
	fmt.Fprintf(&b, "int %s = 0;\n", cnt)
	fmt.Fprintf(&b, "while (%s < %d) {\n", cnt, limit)
	b.WriteString(indent(g.scoped(func() string { return g.loopBody(accName, budget) })))
	fmt.Fprintf(&b, "    %s++;\n", cnt)
	b.WriteString("}")
	return b.String()
}

// loopBody produces a short body that mutates accName so loop output depends on
// iteration count and arithmetic.
func (g *generator) loopBody(accName string, budget int) string {
	ops := []string{"+=", "-=", "*=", "^="}
	op := ops[g.rng.Intn(len(ops))]
	rhs := g.intAtom()
	line := fmt.Sprintf("%s %s %s;", accName, op, rhs)
	if budget > 1 && g.rng.Intn(3) == 0 {
		if extra := g.nestedStatement(budget - 1); extra != "" {
			return line + "\n" + extra
		}
	}
	return line
}

// stringBuilderStmt builds a StringBuilder and records its result for printing.
func (g *generator) stringBuilderStmt() string {
	name := g.freshName(tString) // treat the eventual toString() as a String local
	// Remove it from String locals as a *variable*; instead declare a real SB and
	// assign toString to the String local.
	sb := fmt.Sprintf("sb%d", g.nextID)
	g.nextID++
	var b strings.Builder
	fmt.Fprintf(&b, "StringBuilder %s = new StringBuilder();\n", sb)
	parts := 1 + g.rng.Intn(3)
	for i := 0; i < parts; i++ {
		switch g.rng.Intn(3) {
		case 0:
			fmt.Fprintf(&b, "%s.append(%q);\n", sb, g.word())
		case 1:
			fmt.Fprintf(&b, "%s.append(%s);\n", sb, g.intAtom())
		default:
			fmt.Fprintf(&b, "%s.append(%s);\n", sb, g.boolAtom())
		}
	}
	fmt.Fprintf(&b, "String %s = %s.toString();", name, sb)
	return b.String()
}

// arrayStmt declares a small int array, mutates an element, and records the sum
// for printing (exercises array creation, indexing, and length).
func (g *generator) arrayStmt() string {
	arr := fmt.Sprintf("arr%d", g.nextID)
	g.nextID++
	n := 2 + g.rng.Intn(3)
	vals := make([]string, n)
	for i := range vals {
		vals[i] = g.intAtom()
	}
	sumName := g.freshName(tInt)
	iv := fmt.Sprintf("q%d", g.nextID)
	g.nextID++
	var b strings.Builder
	fmt.Fprintf(&b, "int[] %s = {%s};\n", arr, strings.Join(vals, ", "))
	fmt.Fprintf(&b, "%s[0] = %s[%s.length - 1];\n", arr, arr, arr)
	fmt.Fprintf(&b, "int %s = 0;\n", sumName)
	fmt.Fprintf(&b, "for (int %s = 0; %s < %s.length; %s++) { %s += %s[%s]; }",
		iv, iv, arr, iv, sumName, arr, iv)
	return b.String()
}

// stringMethodStmt declares a String and records the results of several
// supported String instance methods (ROADMAP §2). These operate on strings and
// ints/booleans returned by the methods, so they exercise a supported area that
// is independent of the int-local typing gap.
func (g *generator) stringMethodStmt() string {
	base := g.stringLiteral()
	s := g.freshName(tString)
	var b strings.Builder
	fmt.Fprintf(&b, "String %s = %s;\n", s, base)
	// Pick a couple of method results to print; each becomes its own observable.
	r1 := g.freshName(tInt)
	fmt.Fprintf(&b, "int %s = %s.length();\n", r1, s)
	switch g.rng.Intn(4) {
	case 0:
		r := g.freshName(tString)
		fmt.Fprintf(&b, "String %s = %s.toUpperCase();", r, s)
	case 1:
		r := g.freshName(tString)
		// substring with an in-range index derived from length keeps it valid.
		fmt.Fprintf(&b, "String %s = %s.length() > 1 ? %s.substring(1) : %s;", r, s, s, s)
	case 2:
		r := g.freshName(tBool)
		fmt.Fprintf(&b, "boolean %s = %s.startsWith(%s);", r, s, g.stringLiteral())
	default:
		r := g.freshName(tInt)
		fmt.Fprintf(&b, "int %s = %s.indexOf(%s);", r, s, g.stringLiteral())
	}
	return b.String()
}

// stringLiteral returns a quoted Java string literal drawn from a small pool of
// words, biased toward a few that stress methods (mixed case, leading/trailing
// substrings).
func (g *generator) stringLiteral() string {
	pool := []string{"Hello", "world", "abcabc", "AaBbCc", "", "x", "racecar", "Java"}
	return fmt.Sprintf("%q", pool[g.rng.Intn(len(pool))])
}

// indent prefixes each line of s with four spaces and ensures a trailing newline.
func indent(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		b.WriteString("    ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}
