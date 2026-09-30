package transpiler

import "go/ast"

func init() {
	registerStaticIntrinsic("MessageDigest", "getInstance", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "MessageDigestGetInstance", args...)
	})
	registerStaticIntrinsicResultType("MessageDigest", "getInstance", "MessageDigest")
	for _, method := range []struct{ java, goName, result string }{
		{"update", "Update", "void"}, {"digest", "Digest", "byte[]"},
		{"reset", "Reset", "void"}, {"getAlgorithm", "GetAlgorithm", "String"},
		{"getDigestLength", "GetDigestLength", "int"},
	} {
		registerInstanceIntrinsic("MessageDigest", method.java, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			return selectorCall(recv, method.goName, args)
		})
		registerInstanceIntrinsicResultType("MessageDigest", method.java, method.result)
	}
}
