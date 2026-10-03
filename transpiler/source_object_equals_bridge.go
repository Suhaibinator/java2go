package transpiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Runtime Object equality invokes the exact Java equals(Object) contract. A
// source overload family may have a different Go selector; expose a separate
// execution entry without changing the selectors used by Java overload calls.
func generateSourceObjectEqualsBridgeDecls(ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	if scope == nil || scope.Class == nil || scope.IsInterface || scope.IsEnum {
		return nil
	}
	for _, method := range scope.Methods {
		if !sourceObjectEqualsOverride(scope, method, ctx) {
			continue
		}
		implementation := executionImplementationName(method, scope, ctx)
		if sourceObjectEqualsRuntimeEntryName(implementation) {
			return nil
		}
		bridge := collisionSafeExecutionIdentifier("EqualsJava2goExecution", scope)
		// Reserve actual hidden entries too: they are not source symbol names.
		for generatedIdentifierExists(bridge, scope) || sourceObjectEqualsHiddenEntryOccupied(scope, bridge, ctx) {
			bridge += "1"
		}
		execution := executionNameForClass(scope)
		callCtx := ctx.Clone()
		callCtx.executionContextName = execution
		usedNames := map[string]struct{}{execution: {}}
		for _, name := range scope.GoTypeParameterNames() {
			usedNames[name] = struct{}{}
		}
		receiver := ast.NewIdent(synchronizedUniqueLocalName("receiver", usedNames))
		argument := ast.NewIdent(synchronizedUniqueLocalName("other", usedNames))
		var callReceiver ast.Expr = receiver
		body := []ast.Stmt{&ast.ExprStmt{X: stdjavaCall(callCtx, "ReferenceRequireNonNull", receiver)}}
		if classNeedsVirtualDispatch(scope, ctx) {
			self := ast.NewIdent(synchronizedUniqueLocalName("__java2goEqualsReceiver", usedNames))
			body = append(body,
				&ast.AssignStmt{
					Lhs: []ast.Expr{self}, Tok: token.DEFINE,
					Rhs: []ast.Expr{&ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(classDispatchFieldName(scope))}},
				},
				&ast.IfStmt{
					Cond: &ast.BinaryExpr{X: self, Op: token.EQL, Y: ast.NewIdent("nil")},
					Body: &ast.BlockStmt{List: []ast.Stmt{
						&ast.AssignStmt{Lhs: []ast.Expr{self}, Tok: token.ASSIGN, Rhs: []ast.Expr{receiver}},
					}},
				},
			)
			callReceiver = self
		}
		call := &ast.CallExpr{
			Fun:  &ast.SelectorExpr{X: callReceiver, Sel: ast.NewIdent(implementation)},
			Args: []ast.Expr{ast.NewIdent(execution), argument},
		}
		body = append(body, &ast.ReturnStmt{Results: []ast.Expr{call}})
		return []ast.Decl{&ast.FuncDecl{
			Name: ast.NewIdent(bridge),
			Recv: &ast.FieldList{List: []*ast.Field{{
				Names: []*ast.Ident{receiver},
				Type:  &ast.StarExpr{X: instantiateGenericType(ctx.className, typeParamExprs(scope.GoTypeParameterNames()))},
			}}},
			Type: &ast.FuncType{
				Params: &ast.FieldList{List: []*ast.Field{
					executionParameterField(execution, callCtx),
					{Names: []*ast.Ident{argument}, Type: ast.NewIdent("any")},
				}},
				Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}},
			},
			Body: &ast.BlockStmt{List: body},
		}}
	}
	return nil
}

func sourceObjectEqualsOverride(scope *symbol.ClassScope, method *symbol.Definition, ctx Ctx) bool {
	if method == nil || method.DeclarationNode == nil || method.Constructor || method.IsStatic || method.IsPrivate || method.RequiresHelper || method.OriginalName != "equals" || strings.TrimSpace(method.OriginalType) != "boolean" || len(method.TypeParameters) != 0 || len(method.Parameters) != 1 || executionParameterIsVariadic(method, 0) {
		return false
	}
	declaring := classScopeCtx(scope, ctx)
	declaring.localScope = method
	component, rank := javaArrayTypeParts(definitionParameterJavaSignatureType(method, 0))
	base, _ := parseJavaTypeString(component)
	return rank == 0 && overrideBridgeCanonicalObjectResult(component, base, declaring)
}

func sourceObjectEqualsRuntimeEntryName(name string) bool {
	const base = "EqualsJava2goExecution"
	if !strings.HasPrefix(name, base) {
		return false
	}
	for _, character := range strings.TrimPrefix(name, base) {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func sourceObjectEqualsHiddenEntryOccupied(scope *symbol.ClassScope, name string, ctx Ctx) bool {
	for _, method := range scope.Methods {
		if method != nil && method.DeclarationNode != nil && !method.Constructor && executionImplementationName(method, scope, ctx) == name {
			return true
		}
	}
	return false
}
