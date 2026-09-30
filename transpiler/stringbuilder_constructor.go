package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// Select Java's overload before lowering conversions. A boxed number is an int
// constructor argument through unboxing/widening, never a String seed selected
// from the runtime representation of the value.
func lowerStringBuilderConstructor(_ []ast.Expr, args []ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if len(args) == 0 {
		return stdjavaCall(ctx, "NewStringBuilder")
	}
	if len(args) != 1 {
		return nil
	}
	argument := invocationArgumentNode(invocation, 0)
	actual, known := inferExprJavaType(argument, ctx, source)
	if !known {
		return nil
	}
	if converted, ok := convertJavaValue(args[0], actual, "int", ctx); ok {
		return stdjavaCall(ctx, "NewStringBuilderCapacity", converted)
	}
	if isJavaStringType(actual) || actual == "null" {
		return stdjavaCall(ctx, "NewStringBuilderString", coerceArgumentToExpectedType(args[0], argument, "String", ctx, source))
	}
	// CharSequence requires its own virtual protocol; do not infer that overload
	// from a Go value or silently turn an arbitrary object into text.
	return nil
}
