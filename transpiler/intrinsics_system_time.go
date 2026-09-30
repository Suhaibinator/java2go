package transpiler

import "go/ast"

func init() {
	registerIntrinsicOwner("java.lang.System", true)
	registerStaticIntrinsic("System", "nanoTime", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "SystemNanoTime")
	})
	registerStaticIntrinsicResultType("System", "nanoTime", "long")
	registerStaticIntrinsicImportSignature("System", "nanoTime")
}
