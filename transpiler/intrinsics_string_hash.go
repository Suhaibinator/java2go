package transpiler

import "go/ast"

func init() {
	// String's value hash is already implemented for erased Object receivers.
	// Use that same UTF-16 contract for statically typed calls and method references.
	registerInstanceIntrinsic("String", "hashCode", func(receiver ast.Expr, arguments []ast.Expr, ctx Ctx) ast.Expr {
		if len(arguments) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "ObjectHashCodeExecution", intrinsicExecutionExpr(ctx), receiver)
	})
	registerInstanceIntrinsicResultType("String", "hashCode", "int")
}
