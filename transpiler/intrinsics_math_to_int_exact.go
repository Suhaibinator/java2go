package transpiler

import "go/ast"

func init() {
	registerIntrinsicOwner("java.lang.Math", true)
	registerStaticIntrinsic("Math", "toIntExact", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "MathToIntExact", args[0])
	})
	registerStaticIntrinsicResultType("Math", "toIntExact", "int")
	registerStaticIntrinsicImportSignature("Math", "toIntExact", "long")
}
