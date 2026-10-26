package transpiler

import "go/ast"

func init() {
	registerInstanceIntrinsic("Constructor", "getParameterCount", func(receiver ast.Expr, arguments []ast.Expr, ctx Ctx) ast.Expr {
		if len(arguments) != 0 {
			return nil
		}
		return selectorCall(receiver, "GetParameterCount", nil)
	})
	registerInstanceIntrinsicResultType("Constructor", "getParameterCount", "int")
}
