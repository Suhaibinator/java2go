package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/NickyBoy89/java2go/symbol"
)

// A source SAM keeps its instantiated signature while its canonical interface
// uses the raw Java descriptor. Stage the typed callback once, then adapt at
// the interface boundary: arguments check before entering the callback and a
// covariant result widens only after the callback's own typed checks run.
func genericFamilySAMCallback(value ast.Expr, method *symbol.Definition, scope *symbol.ClassScope, expected string, executionAware bool, ctx Ctx) ast.Expr {
	if method == nil || len(method.TypeParameters) != 0 || canonicalGenericFamily(scope, ctx) == nil {
		return value
	}
	_, bindings := resolveFunctionalInterfaceMethod(ctx, expected)
	physicalBindings := map[string]string{}
	for _, param := range scope.TypeParameters {
		physicalBindings[param.Name] = "Object"
		physicalBindings[param.EmittedName()] = "Object"
	}
	reserved := arrayAllocationReservedNames(value, ctx)
	callbackName := uniqueArrayAllocationName("__java2goFamilyCallback", reserved)
	params := &ast.FieldList{}
	arguments := []ast.Expr{}
	if executionAware {
		executionName := uniqueArrayAllocationName("__java2goFamilyExecution", reserved)
		params.List = append(params.List, executionParameterField(executionName, ctx))
		arguments = append(arguments, ast.NewIdent(executionName))
	}
	changed := false
	for index := range method.Parameters {
		// A spread parameter records its element type in the source symbol, but
		// the SAM descriptor accepts one reified Java array. Adapt that array as
		// a whole so its component check runs before entering the typed callback.
		javaType := definitionParameterJavaSignatureType(method, index)
		physical := substituteJavaTypeParams(javaType, physicalBindings)
		actual := substituteJavaTypeParams(javaType, bindings)
		changed = changed || !javaInferenceSameType(physical, actual, ctx)
		name := ast.NewIdent(uniqueArrayAllocationName("__java2goFamilyArgument"+strconv.Itoa(index), reserved))
		params.List = append(params.List, &ast.Field{Names: []*ast.Ident{name}, Type: javaTypeStringToGoTypeExpr(physical, inScopeTypeParameters(ctx), ctx)})
		argument := projectDirectOwnerErasedView(name, actual, physical, ctx)
		if _, rank := javaArrayTypeParts(actual); rank > 0 && !javaInferenceTypeAssignable(physical, actual, ctx) {
			if descriptor, ok := javaTypeDescriptorExpr(actual, ctx); ok {
				arrayType := javaTypeStringToGoTypeExpr(actual, inScopeTypeParameters(ctx), ctx)
				argument = stdjavaGenericCall(ctx, "JavaArrayCast", []ast.Expr{arrayType}, []ast.Expr{name, descriptor})
			}
		}
		arguments = append(arguments, argument)
	}
	results := &ast.FieldList{}
	if !javaMethodResultIsVoid(method) {
		physical := substituteJavaTypeParams(method.OriginalType, physicalBindings)
		actual := substituteJavaTypeParams(method.OriginalType, bindings)
		changed = changed || !javaInferenceSameType(physical, actual, ctx)
		results.List = append(results.List, &ast.Field{Type: javaTypeStringToGoTypeExpr(physical, inScopeTypeParameters(ctx), ctx)})
	}
	if !changed {
		return value
	}
	callbackType := &ast.FuncType{Params: params, Results: results}
	call := &ast.CallExpr{Fun: ast.NewIdent(callbackName), Args: arguments}
	var invoke ast.Stmt = &ast.ExprStmt{X: call}
	if len(results.List) > 0 {
		invoke = &ast.ReturnStmt{Results: []ast.Expr{call}}
	}
	callback := &ast.FuncLit{Type: callbackType, Body: &ast.BlockStmt{List: []ast.Stmt{invoke}}}
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: callbackType}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{
		&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(callbackName)}, Tok: token.DEFINE, Rhs: []ast.Expr{value}},
		&ast.ReturnStmt{Results: []ast.Expr{callback}},
	}}}}
}
