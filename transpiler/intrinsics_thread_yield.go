package transpiler

import "go/ast"

func registerThreadYieldIntrinsics() {
	// Register the canonical owner before the short dispatch key is used. The
	// normal qualifier resolver retains lexical/source/explicit-import guards.
	registerIntrinsicOwner("java.lang.Thread", true)
	registerStaticIntrinsic("Thread", "yield", func(receiver ast.Expr, arguments []ast.Expr, ctx Ctx) ast.Expr {
		if len(arguments) != 0 {
			return nil
		}
		execution := executionExpr(ctx)
		if execution == nil {
			return nil
		}
		return stdjavaCall(ctx, "ThreadYieldExecution", execution)
	})
	registerStaticIntrinsicResultType("Thread", "yield", "void")
}
