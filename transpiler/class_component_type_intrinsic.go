package transpiler

import "go/ast"

// Canonical owner admission retains source Class declarations and lexical
// binders on their source methods. The fixed result also retains the library
// declaration when another Class operation is chained from this accessor.
func init() {
	registerInstanceIntrinsic("Class", "getComponentType", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "ClassGetComponentTypeExecution", intrinsicExecutionExpr(ctx), recv)
	})
	registerInstanceIntrinsicResultType("Class", "getComponentType", "java.lang.Class")
}
