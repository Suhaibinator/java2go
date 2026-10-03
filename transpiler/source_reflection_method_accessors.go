package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"strconv"
)

// Java visibility is enforced by Method.invoke, independently of Go export
// visibility. A declaring-body accessor permits accessible non-public methods
// to execute without reflect exposing or guessing unexported Go selectors.
func sourceReflectionMethodNeedsAccessor(scope *symbol.ClassScope, method *symbol.Definition, ctx Ctx) bool {
	return sourceReflectionDeclaredMethod(scope, method) && !scope.IsInterface && !scope.IsEnum && !method.IsStatic && !method.RequiresHelper && !ast.IsExported(symbol.GoIdentifier(executionImplementationName(method, scope, ctx)))
}
func sourceReflectionMethodExecutionName(scope *symbol.ClassScope, method *symbol.Definition, ctx Ctx) string {
	if sourceReflectionMethodNeedsAccessor(scope, method, ctx) {
		return collisionSafeExecutionIdentifier("Java2goReflectMethod"+method.Name+executionMethodSuffix, scope)
	}
	return executionImplementationName(method, scope, ctx)
}
func sourceReflectionPrivateMethodAccessors(scope *symbol.ClassScope, ctx Ctx) []ast.Decl {
	var out []ast.Decl
	for _, method := range scope.Methods {
		if !sourceReflectionMethodNeedsAccessor(scope, method, ctx) {
			continue
		}
		declaring := classScopeCtx(scope, ctx).Clone()
		declaring.localScope = method
		used := make(map[string]struct{})
		for _, name := range scope.GoTypeParameterNames() {
			used[name] = struct{}{}
		}
		allocate := func(base string) *ast.Ident {
			name := synchronizedUniqueLocalName(base, used)
			used[name] = struct{}{}
			return ast.NewIdent(name)
		}
		receiver := allocate("receiver")
		execution := allocate("execution")
		parameters := &ast.FieldList{List: []*ast.Field{executionParameterField(execution.Name, ctx)}}
		arguments := []ast.Expr{execution}
		for index, parameter := range method.Parameters {
			name := allocate("argument" + strconv.Itoa(index))
			typ := executionParameterTypeExpr(method, index, parameter.OriginalType, inScopeTypeParameters(declaring), declaring)
			typ = directOwnerTypeParameterMethodParameterType(scope, method, index, typ, declaring)
			parameters.List = append(parameters.List, &ast.Field{Names: []*ast.Ident{name}, Type: typ})
			arguments = append(arguments, name)
		}
		call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(executionImplementationName(method, scope, declaring))}, Args: arguments}
		var results *ast.FieldList
		var body ast.Stmt = &ast.ExprStmt{X: call}
		if !javaMethodResultIsVoid(method) {
			typ := javaTypeStringToGoTypeExpr(method.OriginalType, inScopeTypeParameters(declaring), declaring)
			typ = directOwnerTypeParameterMethodResultType(scope, method, typ, declaring)
			results = &ast.FieldList{List: []*ast.Field{{Type: typ}}}
			body = &ast.ReturnStmt{Results: []ast.Expr{call}}
		}
		out = append(out, &ast.FuncDecl{Name: ast.NewIdent(sourceReflectionMethodExecutionName(scope, method, declaring)), Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{receiver}, Type: classSubobjectPointerTypeExpr(scope, scope.GoTypeParameterNames(), scope, declaring)}}}, Type: &ast.FuncType{Params: parameters, Results: results}, Body: &ast.BlockStmt{List: []ast.Stmt{body}}})
	}
	return out
}
