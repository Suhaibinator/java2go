package transpiler

import "go/ast"

// Class inherits Object's identity-based hashCode and equals. Canonical owner
// admission keeps source declarations and lexical Class binders on source calls.
func init() {
	registerInstanceIntrinsic("Class", "hashCode", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "ObjectHashCodeExecution", intrinsicExecutionExpr(ctx), recv)
	})
	registerInstanceIntrinsicResultType("Class", "hashCode", "int")
	registerInstanceIntrinsic("Class", "equals", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "ObjectEqualsExecution", intrinsicExecutionExpr(ctx), recv, args[0])
	})
	registerInstanceIntrinsicResultType("Class", "equals", "boolean")
}
