package transpiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// fieldResolution retains the class that actually declares a field. Java
// permits inherited static fields to be selected through a subclass, but the
// declaring class owns both the storage and the initialization trigger.
type fieldResolution struct {
	def   *symbol.Definition
	owner *symbol.ClassScope
}

func findFieldResolutionInHierarchy(start *symbol.ClassScope, fieldName string, ctx Ctx) *fieldResolution {
	seen := map[*symbol.ClassScope]struct{}{}
	for scope := start; scope != nil; scope = resolveSuperclassScopeInDeclaringContext(ctx, scope) {
		if _, ok := seen[scope]; ok {
			return nil
		}
		seen[scope] = struct{}{}
		if field := scope.FindFieldByName(fieldName); field != nil {
			return &fieldResolution{def: field, owner: scope}
		}
	}
	return nil
}

type staticFieldAccess struct {
	resolution    *fieldResolution
	qualifierNode *sitter.Node
}

func identifierHasValueBinding(name string, ctx Ctx) bool {
	if ctx.localScope != nil {
		if ctx.localScope.ParameterByName(name) != nil || ctx.localScope.FindVariable(name) != nil {
			return true
		}
	}
	if ctx.currentClass != nil && findFieldInHierarchy(ctx.currentClass, name, ctx) != nil {
		return true
	}
	return false
}

func identifierHasLocalBinding(name string, ctx Ctx) bool {
	return ctx.localScope != nil &&
		(ctx.localScope.ParameterByName(name) != nil || ctx.localScope.FindVariable(name) != nil)
}

func staticFieldQualifierScope(node *sitter.Node, source []byte, ctx Ctx) (scope *symbol.ClassScope, typeQualifier bool) {
	if node == nil {
		return nil, false
	}
	switch node.Type() {
	case "this":
		return ctx.currentClass, false
	case "super":
		return resolveSuperclassScopeInDeclaringContext(ctx, ctx.currentClass), false
	case "identifier":
		name := node.Content(source)
		if !identifierHasValueBinding(name, ctx) {
			if resolved := resolveClassScopeByQualifiedName(ctx, name); resolved != nil {
				return resolved, true
			}
		}
	case "scoped_identifier", "type_identifier", "generic_type":
		if resolved := resolveClassScopeByQualifiedName(ctx, node.Content(source)); resolved != nil {
			return resolved, true
		}
	}

	javaType, ok := inferExprJavaType(node, ctx, source)
	if !ok {
		return nil, false
	}
	base, _ := parseJavaTypeString(javaType)
	return resolveClassScopeByQualifiedName(ctx, base), false
}

// resolveStaticFieldAccess recognizes both unqualified fields and Java's
// class-/expression-qualified form. The qualifier is retained only when Java
// evaluates it as an expression; a type name has no runtime evaluation.
func resolveStaticFieldAccess(node *sitter.Node, source []byte, ctx Ctx) (*staticFieldAccess, bool) {
	if node == nil {
		return nil, false
	}

	switch node.Type() {
	case "identifier":
		name := node.Content(source)
		if identifierHasLocalBinding(name, ctx) {
			return nil, false
		}
		for scope := ctx.currentClass; scope != nil; scope = scope.Enclosing {
			resolution := findFieldResolutionInHierarchy(scope, name, ctx)
			if resolution == nil {
				continue
			}
			if !resolution.def.IsStatic {
				return nil, false
			}
			return &staticFieldAccess{resolution: resolution}, true
		}
		return nil, false

	case "field_access":
		objectNode := node.ChildByFieldName("object")
		fieldNode := node.ChildByFieldName("field")
		if objectNode == nil || fieldNode == nil {
			return nil, false
		}
		owner, typeQualifier := staticFieldQualifierScope(objectNode, source, ctx)
		resolution := findFieldResolutionInHierarchy(owner, fieldNode.Content(source), ctx)
		if resolution == nil || resolution.def == nil || !resolution.def.IsStatic {
			return nil, false
		}
		access := &staticFieldAccess{resolution: resolution}
		if !typeQualifier && objectNode.Type() != "this" && objectNode.Type() != "super" {
			access.qualifierNode = objectNode
		}
		return access, true
	}

	return nil, false
}

func staticFieldStorageExpr(access *staticFieldAccess, ctx Ctx) ast.Expr {
	if access == nil || access.resolution == nil || access.resolution.def == nil {
		return &ast.BadExpr{}
	}
	return qualifiedNameExpr(
		access.resolution.def.Name,
		findJavaPackageForClassScope(access.resolution.owner),
		ctx,
	)
}

func staticFieldValueType(access *staticFieldAccess, ctx Ctx) ast.Expr {
	if access == nil || access.resolution == nil || access.resolution.def == nil {
		return &ast.Ident{Name: "any"}
	}
	javaType := qualifyJavaTypeInDeclaringContext(access.resolution.def.OriginalType, access.resolution.owner)
	typeExpr := javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx)
	return abstractClassToInterface(typeExpr, javaType, ctx)
}

func staticFieldQualifierStmt(access *staticFieldAccess, source []byte, ctx Ctx) ast.Stmt {
	if access == nil || access.qualifierNode == nil {
		return nil
	}
	return &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{ParseExpr(access.qualifierNode, source, ctx)},
	}
}

func staticFieldEnsureStmt(access *staticFieldAccess, ctx Ctx) ast.Stmt {
	if access == nil || access.resolution == nil {
		return nil
	}
	if access.resolution.def != nil && access.resolution.def.IsCompileTimeConstant {
		return nil
	}
	execution := executionExpr(ctx)
	if execution == nil {
		execution = newExecutionExpr(ctx)
	}
	return classInitializationEnsureStmt(access.resolution.owner, execution, ctx)
}

func staticFieldPrelude(access *staticFieldAccess, source []byte, ctx Ctx, initialize bool) []ast.Stmt {
	var body []ast.Stmt
	if qualifier := staticFieldQualifierStmt(access, source, ctx); qualifier != nil {
		body = append(body, qualifier)
	}
	if initialize {
		if ensure := staticFieldEnsureStmt(access, ctx); ensure != nil {
			body = append(body, ensure)
		}
	}
	return body
}

func lowerStaticFieldRead(access *staticFieldAccess, source []byte, ctx Ctx) ast.Expr {
	storage := staticFieldStorageExpr(access, ctx)
	body := staticFieldPrelude(access, source, ctx, true)
	if len(body) == 0 {
		return storage
	}
	body = append(body, &ast.ReturnStmt{Results: []ast.Expr{storage}})
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: staticFieldValueType(access, ctx)}}}},
		Body: &ast.BlockStmt{List: body},
	}}
}

func staticFieldTempName(node *sitter.Node, source []byte, ctx Ctx, base string, already ...string) string {
	used := affineLoopUsedNames(node, source, ctx)
	for _, name := range already {
		used[name] = struct{}{}
	}
	return synchronizedUniqueLocalName(base, used)
}

func typedLocalDeclaration(name string, typeExpr, value ast.Expr) ast.Stmt {
	return &ast.DeclStmt{Decl: &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{&ast.ValueSpec{
			Names:  []*ast.Ident{{Name: name}},
			Type:   typeExpr,
			Values: []ast.Expr{value},
		}},
	}}
}

// lowerStaticFieldAssignment preserves the distinct JVM sequencing of putstatic
// and getstatic. A simple write initializes only after its RHS has completed;
// a compound write initializes and captures the old value before its RHS.
func lowerStaticFieldAssignment(node *sitter.Node, source []byte, ctx Ctx) (ast.Expr, bool) {
	if node == nil || node.Type() != "assignment_expression" || node.ChildCount() < 3 {
		return nil, false
	}
	lhsNode := node.Child(0)
	opNode := node.Child(1)
	rhsNode := node.Child(2)
	access, ok := resolveStaticFieldAccess(lhsNode, source, ctx)
	if !ok || opNode == nil || rhsNode == nil {
		return nil, false
	}

	javaType := qualifyJavaTypeInDeclaringContext(access.resolution.def.OriginalType, access.resolution.owner)
	valueType := staticFieldValueType(access, ctx)
	storage := staticFieldStorageExpr(access, ctx)
	body := staticFieldPrelude(access, source, ctx, false)
	operator := opNode.Content(source)

	if operator == "=" {
		rhsCtx := ctx.Clone()
		rhsCtx.expectedType = javaType
		rhsCtx.expectedTypeRoot = rhsNode
		rhs := ParseExpr(rhsNode, source, rhsCtx)
		rhs = coerceArgumentToExpectedType(rhs, rhsNode, javaType, ctx, source)
		valueName := staticFieldTempName(node, source, ctx, "__java2goStaticFieldValue")
		body = append(body, typedLocalDeclaration(valueName, valueType, rhs))
		if ensure := staticFieldEnsureStmt(access, ctx); ensure != nil {
			body = append(body, ensure)
		}
		body = append(body,
			&ast.AssignStmt{Lhs: []ast.Expr{storage}, Tok: token.ASSIGN, Rhs: []ast.Expr{&ast.Ident{Name: valueName}}},
			&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: valueName}}},
		)
		return &ast.CallExpr{Fun: &ast.FuncLit{
			Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: valueType}}}},
			Body: &ast.BlockStmt{List: body},
		}}, true
	}

	if ensure := staticFieldEnsureStmt(access, ctx); ensure != nil {
		body = append(body, ensure)
	}
	oldName := staticFieldTempName(node, source, ctx, "__java2goStaticFieldOld")
	rhsName := staticFieldTempName(node, source, ctx, "__java2goStaticFieldRHS", oldName)
	valueName := staticFieldTempName(node, source, ctx, "__java2goStaticFieldValue", oldName, rhsName)
	body = append(body, typedLocalDeclaration(oldName, valueType, storage))

	rhsJavaType, known := inferExprJavaType(rhsNode, ctx, source)
	if !known || strings.TrimSpace(rhsJavaType) == "" {
		rhsJavaType = "Object"
	}
	rhsType := javaTypeStringToGoTypeExpr(rhsJavaType, inScopeTypeParameters(ctx), ctx)
	body = append(body, typedLocalDeclaration(rhsName, rhsType, ParseExpr(rhsNode, source, ctx)))
	value, supported := compoundAssignmentValue(
		operator,
		&ast.Ident{Name: oldName},
		&ast.Ident{Name: rhsName},
		javaType,
		rhsJavaType,
		ctx,
	)
	if !supported {
		return &ast.BadExpr{}, true
	}
	body = append(body,
		typedLocalDeclaration(valueName, valueType, value),
		&ast.AssignStmt{Lhs: []ast.Expr{storage}, Tok: token.ASSIGN, Rhs: []ast.Expr{&ast.Ident{Name: valueName}}},
		&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: valueName}}},
	)
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: valueType}}}},
		Body: &ast.BlockStmt{List: body},
	}}, true
}

func lowerStaticFieldUpdate(
	node, operandNode *sitter.Node,
	helper string,
	source []byte,
	ctx Ctx,
) (ast.Expr, bool) {
	access, ok := resolveStaticFieldAccess(operandNode, source, ctx)
	if !ok {
		return nil, false
	}
	call := stdjavaCall(ctx, helper, &ast.UnaryExpr{Op: token.AND, X: staticFieldStorageExpr(access, ctx)})
	body := staticFieldPrelude(access, source, ctx, true)
	if len(body) == 0 {
		return call, true
	}
	body = append(body, &ast.ReturnStmt{Results: []ast.Expr{call}})
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: staticFieldValueType(access, ctx)}}}},
		Body: &ast.BlockStmt{List: body},
	}}, true
}
