package transpiler

import (
	"go/ast"
	"go/token"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func sourceFilterInputStream(scope *symbol.ClassScope, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]bool{}
	for scope != nil && !seen[scope] {
		seen[scope] = true
		base, _ := parseJavaTypeString(scope.Superclass)
		parent := resolveClassScopeByQualifiedName(ctx, base)
		if parent == nil {
			return stripJavaQualifier(base) == "FilterInputStream"
		}
		scope = parent
	}
	return false
}
func filterInputField(ctx Ctx) ast.Expr {
	return &ast.SelectorExpr{X: &ast.SelectorExpr{X: ast.NewIdent(ShortName(ctx.className)), Sel: ast.NewIdent("FilterInputStream")}, Sel: ast.NewIdent("In")}
}
func inheritedFilterInputName(name string, ctx Ctx) bool {
	if name != "in" || !sourceFilterInputStream(ctx.currentClass, ctx) {
		return false
	}
	if ctx.localScope != nil && (ctx.localScope.ParameterByName(name) != nil || ctx.localScope.FindVariable(name) != nil) {
		return false
	}
	return findFieldInHierarchy(ctx.currentClass, name, ctx) == nil
}
func filterInputSuperInvocation(object *sitter.Node, name string, argsNode *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	if !sourceFilterInputStream(ctx.currentClass, ctx) {
		return nil
	}
	isSuper := object.Type() == "super" && resolveClassScopeByQualifiedName(ctx, ctx.currentClass.Superclass) == nil
	isIn := object.Type() == "identifier" && inheritedFilterInputName(object.Content(source), ctx)
	if !isSuper && !isIn {
		return nil
	}
	args := parseArgumentListWithExpectedTypes(argsNode, source, ctx, nil)
	args = append([]ast.Expr{intrinsicExecutionExpr(ctx), filterInputField(ctx)}, args...)
	switch name {
	case "read":
		if len(args) == 2 {
			return stdjavaCall(ctx, "InputStreamReadByteExecution", args...)
		}
		if len(args) == 3 || len(args) == 5 {
			return stdjavaCall(ctx, "InputStreamReadIntoExecution", args...)
		}
	case "close":
		if len(args) == 2 {
			return stdjavaCall(ctx, "InputStreamCloseExecution", args...)
		}
	}
	return nil
}

// Explicit bridge selectors avoid confusing Java's overloaded read methods
// with Go's io.Reader.Read, while preserving the exact source override chosen.
func generateInputStreamBridgeDecls(ctx Ctx) []ast.Decl {
	if !sourceFilterInputStream(ctx.currentClass, ctx) {
		return nil
	}
	result := []ast.Decl{}
	for _, arity := range []int{0, 3} {
		name, runtime := "JavaInputStreamReadByte", "InputStreamReadByteExecution"
		params := []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("execution")}, Type: &ast.StarExpr{X: stdjavaQualifiedExpr("Execution", ctx)}}}
		args := []ast.Expr{ast.NewIdent("execution")}
		if arity == 3 {
			name, runtime = "JavaInputStreamRead", "InputStreamReadIntoExecution"
			params = append(params, &ast.Field{Names: []*ast.Ident{ast.NewIdent("buffer")}, Type: &ast.StarExpr{X: &ast.IndexExpr{X: stdjavaQualifiedExpr("PrimitiveArray", ctx), Index: ast.NewIdent("int8")}}}, &ast.Field{Names: []*ast.Ident{ast.NewIdent("offset"), ast.NewIdent("length")}, Type: ast.NewIdent("int32")})
			args = append(args, ast.NewIdent("buffer"), ast.NewIdent("offset"), ast.NewIdent("length"))
		}
		var selected *symbol.Definition
		var owner *symbol.ClassScope
		for scope := ctx.currentClass; scope != nil; scope = resolveClassScopeByQualifiedName(ctx, scope.Superclass) {
			for _, method := range scope.Methods {
				if method.OriginalName == "read" && !method.IsStatic && len(method.Parameters) == arity {
					if arity == 3 && (method.Parameters[0].OriginalType != "byte[]" || method.Parameters[1].OriginalType != "int" || method.Parameters[2].OriginalType != "int") {
						continue
					}
					selected, owner = method, scope
					break
				}
			}
			if selected != nil {
				break
			}
		}
		var call ast.Expr
		recv := ast.NewIdent(ShortName(ctx.className))
		if selected != nil {
			call = &ast.CallExpr{Fun: &ast.SelectorExpr{X: recv, Sel: ast.NewIdent(executionImplementationName(selected, owner))}, Args: args}
		} else {
			all := append([]ast.Expr{args[0], filterInputField(ctx)}, args[1:]...)
			call = stdjavaCall(ctx, runtime, all...)
		}
		result = append(result, &ast.FuncDecl{Name: ast.NewIdent(name), Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{recv}, Type: &ast.StarExpr{X: instantiateGenericType(ctx.className, typeParamExprs(ctx.currentClass.GoTypeParameterNames()))}}}}, Type: &ast.FuncType{Params: &ast.FieldList{List: params}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("int32")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}}})
	}
	return result
}
func filterInputSuperConstructor(args []ast.Expr, ctx Ctx) ast.Stmt {
	if ctx.currentClass == nil {
		return nil
	}
	base, _ := parseJavaTypeString(ctx.currentClass.Superclass)
	if stripJavaQualifier(base) != "FilterInputStream" || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return nil
	}
	return &ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent(ShortName(ctx.className)), Sel: ast.NewIdent("FilterInputStream")}}, Tok: token.ASSIGN, Rhs: []ast.Expr{stdjavaCall(ctx, "NewFilterInputStream", args...)}}
}
