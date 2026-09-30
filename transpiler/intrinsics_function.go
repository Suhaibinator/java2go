package transpiler

import "go/ast"

func init() { registerIntrinsicOwner("java.util.function.Function", true) }

func isExternalFunctionType(javaType string, ctx Ctx) bool {
	owner, known := canonicalIntrinsicOwner(javaType, ctx)
	return known && owner == "java.util.function.Function"
}
func functionRuntimeTypeExpr(javaType string, arguments, parameters []string, ctx Ctx) (ast.Expr, bool) {
	if !isExternalFunctionType(javaType, ctx) {
		return nil, false
	}
	if len(arguments) == 0 {
		arguments = []string{"Object", "Object"}
	}
	if len(arguments) != 2 {
		return nil, false
	}
	return applyTypeArguments(stdjavaQualifiedExpr("Function", ctx), []ast.Expr{
		javaTypeStringToGoTypeExpr(arguments[0], parameters, ctx),
		javaTypeStringToGoTypeExpr(arguments[1], parameters, ctx),
	}), true
}
func functionCallbackExpr(value ast.Expr, javaType string, ctx Ctx) ast.Expr {
	if !isExternalFunctionType(javaType, ctx) {
		return value
	}
	_, arguments := parseJavaTypeString(javaType)
	if len(arguments) != 2 {
		return value
	}
	types := []ast.Expr{javaTypeStringToGoTypeExpr(arguments[0], inScopeTypeParameters(ctx), ctx), javaTypeStringToGoTypeExpr(arguments[1], inScopeTypeParameters(ctx), ctx)}
	return stdjavaGenericCall(ctx, "FunctionCallbackExecution", types, []ast.Expr{intrinsicExecutionExpr(ctx), value})
}

// isFunctionCallbackExpr recognizes the runtime ABI produced by
// functionCallbackExpr. Its result has one Java parameter; the Execution and
// Function arguments of the adapter call are not callable parameters.
func isFunctionCallbackExpr(value ast.Expr, ctx Ctx) bool {
	call, ok := value.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		return false
	}
	indexed, ok := call.Fun.(*ast.IndexListExpr)
	if !ok || len(indexed.Indices) != 2 {
		return false
	}
	selector, ok := indexed.X.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "FunctionCallbackExecution" {
		return false
	}
	owner, ok := selector.X.(*ast.Ident)
	return ok && owner.Name == markStdjavaUsage(ctx)
}
