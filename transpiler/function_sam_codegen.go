package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// Keep the contract in the implementing declaration's type-parameter context.
// Only declared superclass/interface edges participate in this search.
func sourceFunctionContract(scope *symbol.ClassScope, ctx Ctx) []string {
	if scope == nil {
		return nil
	}
	seen := map[*symbol.ClassScope]bool{}
	var visit func(*symbol.ClassScope, []string) []string
	visit = func(owner *symbol.ClassScope, arguments []string) []string {
		if owner == nil || seen[owner] {
			return nil
		}
		seen[owner] = true
		ownerCtx := classHeaderTypeCtx(owner, ctx)
		bindings := map[string]string{}
		for index, parameter := range owner.TypeParameters {
			if index < len(arguments) {
				bindings[parameter.Name] = arguments[index]
				bindings[parameter.EmittedName()] = arguments[index]
			}
		}
		parents := append(append([]string(nil), owner.ImplementedInterfaces...), owner.Superclass)
		for _, parent := range parents {
			qualified := qualifyDeclaredReferenceType(symbol.JavaType{Original: parent}, ownerCtx)
			qualified = substituteJavaTypeParams(qualified, bindings)
			base, args := parseJavaTypeString(qualified)
			if isExternalFunctionType(base, ownerCtx) {
				if len(args) == 0 {
					args = []string{"Object", "Object"}
				}
				if len(args) == 2 {
					return args
				}
			}
			if result := visit(resolveClassScopeByQualifiedName(ownerCtx, base), args); result != nil {
				return result
			}
		}
		return nil
	}
	return visit(scope, scope.GoTypeParameterNames())
}

// The Java apply overload can have a different generated selector because a
// subclass introduces an unrelated overload. Expose the runtime SAM descriptor
// by forwarding to that exact method family, never by selecting on name alone.
func generateFunctionSAMBridgeDecls(ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	contract := sourceFunctionContract(scope, ctx)
	if len(contract) != 2 || scope.IsInterface || len(sourceNativeFunctionalContracts(scope, ctx)) > 1 {
		return nil
	}
	var method *symbol.Definition
	var owner *symbol.ClassScope
	var actualResult string
	for candidateOwner := scope; candidateOwner != nil; candidateOwner = resolveSuperclassScopeInDeclaringContext(ctx, candidateOwner) {
		arguments := mapClassTypeArgumentStringsToAncestor(scope, scope.GoTypeParameterNames(), candidateOwner, ctx)
		bindings := map[string]string{}
		for index, parameter := range candidateOwner.TypeParameters {
			if index < len(arguments) {
				bindings[parameter.Name] = arguments[index]
				bindings[parameter.EmittedName()] = arguments[index]
			}
		}
		for _, candidate := range candidateOwner.Methods {
			if candidate == nil || candidate.OriginalName != "apply" || candidate.IsStatic || candidate.IsPrivate || len(candidate.Parameters) != 1 || len(candidate.TypeParameters) != 0 {
				continue
			}
			parameter := substituteJavaTypeParams(qualifyJavaTypeInDeclaringContext(definitionParameterJavaSignatureType(candidate, 0), candidateOwner), bindings)
			if !javaInferenceSameType(parameter, contract[0], ctx) {
				continue
			}
			method, owner = candidate, candidateOwner
			actualResult = substituteJavaTypeParams(qualifyJavaTypeInDeclaringContext(candidate.OriginalType, candidateOwner), bindings)
			break
		}
		if method != nil {
			break
		}
	}
	if method == nil || !method.HasBody {
		return nil
	}
	implementation := executionImplementationName(method, owner, ctx)
	var declarations []ast.Decl
	for _, executionAware := range []bool{false, true} {
		name := "Apply"
		if executionAware {
			name += executionMethodSuffix
		}
		if !executionAware && method.Name == name && javaInferenceSameType(actualResult, contract[1], ctx) {
			continue
		}
		if executionAware && implementation == name && javaInferenceSameType(actualResult, contract[1], ctx) {
			continue
		}
		receiver := ast.NewIdent("receiver")
		execution := ast.Expr(stdjavaCall(ctx, "NewExecution"))
		params := &ast.FieldList{}
		if executionAware {
			execution = ast.NewIdent("execution")
			params.List = append(params.List, executionParameterField("execution", ctx))
		}
		params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent("value")}, Type: javaTypeStringToGoTypeExpr(contract[0], inScopeTypeParameters(ctx), ctx)})
		call := ast.Expr(&ast.CallExpr{Fun: &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(implementation)}, Args: []ast.Expr{execution, ast.NewIdent("value")}})
		if actualResult != contract[1] {
			if converted, ok := convertJavaValue(call, actualResult, contract[1], ctx); ok {
				call = converted
			}
		}
		declarations = append(declarations, &ast.FuncDecl{
			Name: ast.NewIdent(name),
			Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{receiver}, Type: &ast.StarExpr{X: instantiateGenericType(ctx.className, typeParamExprs(scope.GoTypeParameterNames()))}}}},
			Type: &ast.FuncType{Params: params, Results: &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(contract[1], inScopeTypeParameters(ctx), ctx)}}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}},
		})
	}
	return declarations
}
