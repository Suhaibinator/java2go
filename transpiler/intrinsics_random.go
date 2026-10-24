package transpiler

import "go/ast"

func init() {
	registerStaticIntrinsic("Collections", "shuffle", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) < 1 || len(args) > 2 {
			return nil
		}
		return stdjavaCall(ctx, "ShuffleList", args...)
	})
	registerStaticIntrinsicResultType("Collections", "shuffle", "void")

	registerConstructorIntrinsic("Random", func(types, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) > 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewRandom", args...)
	})
	for _, method := range []struct{ java, goName, result string }{
		{"setSeed", "SetSeed", "void"}, {"nextInt", "NextInt", "int"},
		{"nextLong", "NextLong", "long"}, {"nextBoolean", "NextBoolean", "boolean"},
		{"nextFloat", "NextFloat", "float"}, {"nextDouble", "NextDouble", "double"},
		{"nextBytes", "NextBytes", "void"},
	} {
		registerInstanceIntrinsic("Random", method.java, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr { // Java 17+ adds origin/bound overloads; until modeled, leave them
			// unsupported rather than silently using the first bound.
			if method.java == "nextInt" && len(args) > 1 {
				return nil
			}
			return selectorCall(recv, method.goName, args)
		})
		registerInstanceIntrinsicResultType("Random", method.java, method.result)
	}
}
