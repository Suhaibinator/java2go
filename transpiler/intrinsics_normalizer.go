package transpiler

import "go/ast"

func init() {
	for _, name := range []string{"NFC", "NFD", "NFKC", "NFKD"} {
		name := name
		registerStaticFieldIntrinsic("Normalizer.Form", name, func(ctx Ctx) ast.Expr { return stdjavaQualifiedExpr("Normalizer"+name, ctx) })
		registerStaticFieldIntrinsicResultType("Normalizer.Form", name, "java.text.Normalizer.Form")
	}
	for _, method := range []struct{ java, runtime, result string }{
		{"normalize", "NormalizerNormalize", "String"}, {"isNormalized", "NormalizerIsNormalized", "boolean"},
	} {
		runtime := method.runtime
		registerStaticIntrinsic("Normalizer", method.java, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 2 {
				return nil
			}
			return stdjavaCall(ctx, runtime, append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args...)...)
		})
		registerStaticIntrinsicResultType("Normalizer", method.java, method.result)
	}
}
