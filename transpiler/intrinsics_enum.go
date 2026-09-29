package transpiler

import "go/ast"

func enumFinalMethod(name string) bool {
	switch name {
	case "name", "ordinal", "getDeclaringClass", "compareTo":
		return true
	}
	return false
}
func init() {
	registerIntrinsicOwner("java.lang.Enum", true)
	for _, spec := range []struct {
		java, helper, result string
		arity                int
	}{
		{"name", "EnumName", "String", 0}, {"ordinal", "EnumOrdinal", "int", 0},
		{"getDeclaringClass", "EnumGetDeclaringClass", "Class", 0}, {"compareTo", "EnumCompareTo", "int", 1},
	} {
		registerInstanceIntrinsic("Enum", spec.java, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != spec.arity {
				return nil
			}
			return stdjavaCall(ctx, spec.helper, append([]ast.Expr{recv}, args...)...)
		})
		registerInstanceIntrinsicResultType("Enum", spec.java, spec.result)
	}
	registerInstanceIntrinsic("Enum", "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "StringValueOfExecution", intrinsicExecutionExpr(ctx), stdjavaCall(ctx, "ReferenceRequireNonNull", recv))
	})
	registerInstanceIntrinsicResultType("Enum", "toString", "String")
}
