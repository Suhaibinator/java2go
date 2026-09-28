package transpiler

import "go/ast"

func init() {
	registerStaticIntrinsic("Character", "digit", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 2 {
			return nil
		}
		return stdjavaCall(ctx, "CharDigit", args...)
	})
	registerStaticIntrinsicResultType("Character", "digit", "int")
	registerStaticIntrinsic("Character", "forDigit", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 2 {
			return nil
		}
		return stdjavaCall(ctx, "CharForDigit", args...)
	})
	registerStaticIntrinsicResultType("Character", "forDigit", "char")

	registerInstanceIntrinsic("CharSequence", "length", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "CharSequenceLength", intrinsicExecutionExpr(ctx), recv)
	})
	registerInstanceIntrinsicResultType("CharSequence", "length", "int")
	registerInstanceIntrinsic("CharSequence", "charAt", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "CharSequenceCharAt", intrinsicExecutionExpr(ctx), recv, args[0])
	})
	registerInstanceIntrinsicResultType("CharSequence", "charAt", "char")
	registerInstanceIntrinsic("String", "regionMatches", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) == 4 {
			args = append([]ast.Expr{&ast.Ident{Name: "false"}}, args...)
		}
		if len(args) != 5 {
			return nil
		}
		return stdjavaCall(ctx, "StringRegionMatches", append([]ast.Expr{recv}, args...)...)
	})
	registerInstanceIntrinsicResultType("String", "regionMatches", "boolean")

}
