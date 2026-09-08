package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// generatedVarargsArrayLiteral is the array allocation required at a Java
// variable-arity call site. Keeping the allocation outside the callee preserves
// fixed-array aliases and lets the runtime enforce the actual component type.
func generatedVarargsArrayLiteral(elementJavaType string, elements []ast.Expr, ctx Ctx) ast.Expr {
	elementType := javaTypeStringToGoTypeExpr(elementJavaType, inScopeTypeParameters(ctx), ctx)
	elementType = abstractClassToInterface(elementType, elementJavaType, ctx)
	descriptor, ok := javaTypeDescriptorExpr(elementJavaType, ctx)
	if !ok {
		descriptor = stdjavaQualifiedExpr("ObjectTypeID", ctx)
	}
	helper := "ReferenceArrayLiteralOf"
	if _, primitive := javaPrimitiveArrayComponent(elementJavaType + "[]"); primitive {
		helper = "PrimitiveArrayLiteral"
	}
	return stdjavaGenericCall(ctx, helper, []ast.Expr{elementType}, append([]ast.Expr{descriptor}, elements...))
}

// generatedMethodReferenceVarargsArguments adapts the SAM's fixed parameter
// list to a Java variable-arity target. An array-compatible final SAM parameter
// is forwarded intact; scalar SAM parameters become a new array for each call.
func generatedMethodReferenceVarargsArguments(
	resolution *methodResolution,
	classTypeArguments []string,
	args []ast.Expr,
	unbound bool,
	ctx Ctx,
) []ast.Expr {
	if resolution == nil || resolution.def == nil || len(resolution.def.Parameters) == 0 {
		return args
	}
	last := len(resolution.def.Parameters) - 1
	if !executionParameterIsVariadic(resolution.def, last) || len(args) < last {
		return args
	}
	sam, bindings := resolveFunctionalInterfaceMethod(ctx, ctx.expectedType)
	if sam == nil {
		return args
	}
	offset := 0
	if unbound {
		offset = 1
	}
	actualTypes := make([]string, len(args))
	for index := range args {
		if index+offset < len(sam.Parameters) {
			actualTypes[index] = substituteJavaTypeParams(definitionParameterJavaSignatureType(sam, index+offset), bindings)
		}
	}
	elementType := instantiatedVarargsElementJavaType(resolution, classTypeArguments, nil)
	if len(args) == len(resolution.def.Parameters) && javaInferenceTypeAssignable(actualTypes[last], elementType+"[]", ctx) {
		return args
	}
	converted := append([]ast.Expr(nil), args...)
	for index := last; index < len(converted); index++ {
		if value, ok := convertJavaValue(converted[index], actualTypes[index], elementType, ctx); ok {
			converted[index] = value
		}
	}
	return append(converted[:last], generatedVarargsArrayLiteral(elementType, converted[last:], ctx))
}

func generatedStaticVarargsMethodReference(
	method ast.Expr,
	definition *symbol.Definition,
	owner *symbol.ClassScope,
	functionType *ast.FuncType,
	executionAware bool,
	ctx Ctx,
) ast.Expr {
	if definition == nil || functionType == nil || len(definition.Parameters) == 0 ||
		!executionParameterIsVariadic(definition, len(definition.Parameters)-1) {
		return method
	}
	params := cloneFieldList(functionType.Params)
	args := methodCallArgs(params)
	if len(args) == 0 {
		return method
	}
	javaArgs := generatedMethodReferenceVarargsArguments(&methodResolution{def: definition, owner: owner}, nil, args[1:], false, ctx)
	if executionAware {
		javaArgs = append([]ast.Expr{args[0]}, javaArgs...)
	}
	call := &ast.CallExpr{Fun: method, Args: javaArgs}
	return &ast.FuncLit{
		Type: &ast.FuncType{Params: params, Results: cloneFieldList(functionType.Results)},
		Body: &ast.BlockStmt{List: []ast.Stmt{invocationClosureCallStatement(call, functionType.Results)}},
	}
}
