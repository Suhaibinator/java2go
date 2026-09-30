package transpiler

import "go/ast"

// A later invocation can mutate an earlier concatenation operand. Keeping the
// earlier formatting call nested preserves Go's lexical call order; flattening
// its arguments would defer plain field reads until after that invocation.
func concatOperandInvokesCode(expr ast.Expr) bool {
	invokes := false
	ast.Inspect(expr, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok && !javaGoExpressionIsTypeConversion(call.Fun) {
			invokes = true
		}
		return !invokes
	})
	return invokes
}
