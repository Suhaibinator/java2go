package fuzz

import (
	"fmt"
	"strings"
)

// expr generates a Java expression of the requested type. depth bounds nesting;
// at depth 0 it returns an atom (literal or in-scope local).
func (g *generator) expr(t jType, depth int) string {
	if depth <= 0 {
		return g.atom(t)
	}
	switch t {
	case tInt:
		return g.intExpr(depth)
	case tLong:
		return g.longExpr(depth)
	case tDouble:
		return g.doubleExpr(depth)
	case tBool:
		return g.boolExpr(depth)
	case tChar:
		return g.charExpr(depth)
	case tString:
		return g.stringExpr(depth)
	}
	return g.atom(t)
}

// intExpr is the most edge-case-dense generator: arithmetic that overflows,
// shifts with large counts, char/long casts back to int, and ++/-- in
// expression position.
func (g *generator) intExpr(depth int) string {
	if depth <= 0 {
		return g.intAtom()
	}
	switch g.weighted([]int{20, 14, 14, 10, 10, 8, 8, 8, 8}) {
	case 0: // binary arithmetic, possibly overflowing
		op := pick(g.rng, "+", "-", "*")
		return g.paren(g.intExpr(depth-1)) + " " + op + " " + g.paren(g.intExpr(depth-1))
	case 1: // division/modulo with a guaranteed non-zero divisor
		op := pick(g.rng, "/", "%")
		return g.paren(g.intExpr(depth-1)) + " " + op + " " + g.nonZeroIntAtom()
	case 2: // shift with a count that may meet/exceed the 32-bit width
		op := pick(g.rng, "<<", ">>", ">>>")
		return g.paren(g.intExpr(depth-1)) + " " + op + " " + g.shiftAmount()
	case 3: // bitwise
		op := pick(g.rng, "&", "|", "^")
		return g.paren(g.intExpr(depth-1)) + " " + op + " " + g.intAtom()
	case 4: // unary
		op := pick(g.rng, "-", "~")
		return g.unary(op, g.intExpr(depth-1))
	case 5: // char promoted to int via arithmetic
		return g.charAtom() + " + " + g.intAtom()
	case 6: // narrowing cast from long
		return "(int) (" + g.longExpr(depth-1) + ")"
	case 7: // ++/-- in expression position on an int local
		if e, ok := g.incDecExpr(); ok {
			return e
		}
		return g.intAtom()
	case 8: // ternary
		return g.paren(g.boolExpr(1)) + " ? " + g.intAtom() + " : " + g.intAtom()
	}
	return g.intAtom()
}

func (g *generator) longExpr(depth int) string {
	if depth <= 0 {
		return g.longAtom()
	}
	switch g.weighted([]int{24, 16, 16, 12, 12, 10}) {
	case 0:
		op := pick(g.rng, "+", "-", "*")
		return g.paren(g.longExpr(depth-1)) + " " + op + " " + g.paren(g.longExpr(depth-1))
	case 1:
		op := pick(g.rng, "/", "%")
		return g.paren(g.longExpr(depth-1)) + " " + op + " " + g.nonZeroLongAtom()
	case 2: // long shift: counts up to 63+ exercise the 6-bit mask
		op := pick(g.rng, "<<", ">>", ">>>")
		return g.paren(g.longExpr(depth-1)) + " " + op + " " + g.longShiftAmount()
	case 3:
		op := pick(g.rng, "&", "|", "^")
		return g.paren(g.longExpr(depth-1)) + " " + op + " " + g.longAtom()
	case 4: // widening int into long context
		return "(long) " + g.paren(g.intExpr(depth-1))
	case 5:
		return g.unary("-", g.longExpr(depth-1))
	}
	return g.longAtom()
}

func (g *generator) doubleExpr(depth int) string {
	if depth <= 0 {
		return g.doubleAtom()
	}
	switch g.weighted([]int{30, 18, 16, 12}) {
	case 0:
		op := pick(g.rng, "+", "-", "*")
		return g.paren(g.doubleExpr(depth-1)) + " " + op + " " + g.paren(g.doubleExpr(depth-1))
	case 1:
		return g.paren(g.doubleExpr(depth-1)) + " / " + g.nonZeroDoubleAtom()
	case 2: // int/double mixed promotes to double
		return g.intAtom() + " / " + g.nonZeroDoubleAtom()
	case 3:
		return g.unary("-", g.doubleExpr(depth-1))
	}
	return g.doubleAtom()
}

func (g *generator) boolExpr(depth int) string {
	if depth <= 0 {
		return g.boolAtom()
	}
	switch g.weighted([]int{26, 22, 16, 10, 10}) {
	case 0: // int comparison
		op := pick(g.rng, "<", "<=", ">", ">=", "==", "!=")
		return g.paren(g.intExpr(depth-1)) + " " + op + " " + g.intAtom()
	case 1: // boolean combinator
		op := pick(g.rng, "&&", "||")
		return g.paren(g.boolExpr(depth-1)) + " " + op + " " + g.paren(g.boolExpr(depth-1))
	case 2:
		return "!" + g.paren(g.boolExpr(depth-1))
	case 3: // long comparison
		op := pick(g.rng, "<", ">", "==")
		return g.paren(g.longExpr(depth-1)) + " " + op + " " + g.longAtom()
	case 4:
		return g.boolAtom()
	}
	return g.boolAtom()
}

func (g *generator) charExpr(depth int) string {
	if depth <= 0 {
		return g.charAtom()
	}
	switch g.weighted([]int{40, 30, 30}) {
	case 0:
		return g.charAtom()
	case 1: // char produced by casting an int sum (wraps to 16-bit)
		return "(char) (" + g.charAtom() + " + " + g.intAtom() + ")"
	case 2: // char cast from a possibly-large int
		return "(char) (" + g.intExpr(depth-1) + ")"
	}
	return g.charAtom()
}

// stringExpr builds string + numeric concatenations, the classic left-to-right
// associativity trap ("a" + 1 + 2 vs "a" + (1 + 2)).
func (g *generator) stringExpr(depth int) string {
	if depth <= 0 {
		return fmt.Sprintf("%q", g.word())
	}
	switch g.weighted([]int{30, 24, 20, 14, 12}) {
	case 0:
		return fmt.Sprintf("%q", g.word())
	case 1: // string + number (left assoc): concatenation
		return fmt.Sprintf("%q + %s", g.word(), g.intAtom())
	case 2: // number + number + string: arithmetic THEN concat
		return fmt.Sprintf("%s + %s + %q", g.intAtom(), g.intAtom(), g.word())
	case 3: // string + (number + number): parenthesized arithmetic
		return fmt.Sprintf("%q + (%s + %s)", g.word(), g.intAtom(), g.intAtom())
	case 4: // string + char and string + bool
		return fmt.Sprintf("%q + %s + %s", g.word(), g.charAtom(), g.boolAtom())
	}
	return fmt.Sprintf("%q", g.word())
}

// incDecExpr returns a pre/post increment expression over an existing int local,
// or false if none is in scope.
func (g *generator) incDecExpr() (string, bool) {
	_, name, ok := g.pickIntLocal()
	if !ok {
		return "", false
	}
	switch g.rng.Intn(4) {
	case 0:
		return name + "++", true
	case 1:
		return name + "--", true
	case 2:
		return "++" + name, true
	default:
		return "--" + name, true
	}
}

// paren wraps an expression in parentheses unless it is already a simple atom,
// keeping nested arithmetic unambiguous.
func (g *generator) paren(e string) string {
	if isAtom(e) {
		return e
	}
	return "(" + e + ")"
}

// unary applies a prefix operator, always parenthesizing the operand so a
// leading sign on the operand can never fuse into "--", "+-", or "~-" (which
// Java would parse as decrement/something other than intended).
func (g *generator) unary(op, operand string) string {
	return op + "(" + operand + ")"
}

func isAtom(e string) bool {
	return !strings.ContainsAny(e, " ")
}
