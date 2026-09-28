package transpiler

import "go/ast"

func reflectRuntimeTypeExpr(baseName string, ctx Ctx) (ast.Expr, bool) {
	if baseName == "Type" && resolveClassScopeByQualifiedName(ctx, baseName) == nil {
		return stdjavaQualifiedExpr("ReflectType", ctx), true
	}
	return nil, false
}

func init() {
	for _, receiver := range []string{"Type", "Class"} {
		registerInstanceIntrinsic(receiver, "getTypeName", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "ReflectTypeNameExecution", intrinsicExecutionExpr(ctx), recv)
		})
		registerInstanceIntrinsicResultType(receiver, "getTypeName", "String")
		registerInstanceIntrinsic(receiver, "getClass", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "ObjectGetClass", recv)
		})
		registerInstanceIntrinsicResultType(receiver, "getClass", "Class")
	}
}
