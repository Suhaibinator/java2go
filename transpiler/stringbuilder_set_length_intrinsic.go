package transpiler

import "go/ast"

func init() {
	for _, typeName := range []string{"StringBuilder", "StringBuffer"} {
		registerInstanceIntrinsic(typeName, "setLength", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			return methodCall(recv, "SetLength", args[0])
		})
		registerInstanceIntrinsicResultType(typeName, "setLength", "void")
	}
}
