package transpiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Source declarations determine volatility. Display names and source/library
// spellings never select this lowering; inherited fields retain their owner.
func volatileFieldDefinition(field *symbol.Definition) bool {
	if field == nil || field.DeclarationNode == nil {
		return false
	}
	for _, child := range nodeutil.NamedChildrenOf(field.DeclarationNode) {
		if child.Type() != "modifiers" {
			continue
		}
		for _, modifier := range nodeutil.UnnamedChildrenOf(child) {
			if modifier.Type() == "volatile" {
				return true
			}
		}
	}
	return false
}

// Local and anonymous classes are hoisted during expression emission, after
// initial source scopes are assembled. Walk the original declaration tree so
// they request reflection/identity metadata before hoisting begins.
func sourceNodeDeclaresVolatileField(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	if node.Type() == "field_declaration" && volatileFieldDefinition(&symbol.Definition{DeclarationNode: node}) {
		return true
	}
	for _, child := range nodeutil.NamedChildrenOf(node) {
		if sourceNodeDeclaresVolatileField(child) {
			return true
		}
	}
	return false
}

func volatileFieldBackingType(field *symbol.Definition, ordinary ast.Expr, ctx Ctx) ast.Expr {
	if !volatileFieldDefinition(field) {
		return ordinary
	}
	return stdjavaQualifiedExpr("VolatileFieldCell", ctx)
}

func volatileFieldLoad(storage, valueType ast.Expr, ctx Ctx, field *symbol.Definition, owner *symbol.ClassScope) ast.Expr {
	if selector, ok := storage.(*ast.SelectorExpr); ok && field != nil && !field.IsStatic {
		storage = &ast.SelectorExpr{X: stdjavaCall(ctx, "ReferenceRequireNonNull", selector.X), Sel: selector.Sel}
	}
	args := []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: storage}}
	args = volatileLoadTypeArgument(args, field, owner, ctx)
	return stdjavaGenericCall(ctx, "VolatileLoad", []ast.Expr{valueType}, args)
}

func volatileLoadTypeArgument(args []ast.Expr, field *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) []ast.Expr {
	if field == nil {
		return args
	}
	if _, primitive := javaPrimitiveType(field.OriginalType); primitive {
		return args
	}
	// Runtime assignability follows the declaration's Java erasure, while the
	// generic Go type argument retains the consuming view's physical ABI.
	declaring := classScopeCtx(sourceReflectionJavaDeclarationScope(owner), ctx)
	declaring.localScope = field
	declaring.syntheticTypeParameters = nil
	javaType := qualifyDeclaredReferenceType(symbol.JavaType{
		Original: field.OriginalType, TypeParameterBindings: field.TypeParameterBindings,
	}, declaring)
	if id, ok := javaTypeDescriptorExpr(javaType, declaring); ok {
		args = append(args, id)
	}
	return args
}

// Used by initialization and metadata, which already hold an exact backing
// selector. It never re-resolves a display name or allocates replacement storage.
func volatileFieldStoreStmt(field *symbol.Definition, storage, value ast.Expr, ctx Ctx) ast.Stmt {
	if !volatileFieldDefinition(field) {
		return &ast.AssignStmt{Lhs: []ast.Expr{storage}, Tok: token.ASSIGN, Rhs: []ast.Expr{value}}
	}
	// Explicit type arguments are required for null and untyped constants. The
	// physical type is the same declaration-bound erased ABI as reflection.
	valueType := sourceReflectionFieldStorageType(ctx.currentClass, field, ctx)
	return &ast.ExprStmt{X: stdjavaGenericCall(ctx, "VolatileStore", []ast.Expr{valueType}, []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: storage}, value})}
}

type volatileFieldAccess struct {
	resolution *fieldResolution
	static     *staticFieldAccess
}

func resolveVolatileFieldAccess(node *sitter.Node, source []byte, ctx Ctx) (*volatileFieldAccess, bool) {
	if node == nil {
		return nil, false
	}
	if node.Type() == "parenthesized_expression" && node.NamedChildCount() == 1 {
		return resolveVolatileFieldAccess(node.NamedChild(0), source, ctx)
	}
	if access, ok := resolveStaticFieldAccess(node, source, ctx); ok {
		if volatileFieldDefinition(access.resolution.def) {
			return &volatileFieldAccess{resolution: access.resolution, static: access}, true
		}
		return nil, false
	}
	var resolution *fieldResolution
	switch node.Type() {
	case "identifier":
		name := node.Content(source)
		if identifierHasLocalBinding(name, ctx) {
			return nil, false
		}
		if ctx.localScope != nil && ctx.localScope.IsStatic {
			return nil, false
		}
		resolution = findFieldResolutionInHierarchy(ctx.currentClass, name, ctx)
		if resolution == nil {
			for scope := ctx.currentClass; scope != nil && scope.IsInner; scope = scope.Enclosing {
				resolution = findFieldResolutionInHierarchy(scope.Enclosing, name, ctx)
				if resolution != nil {
					break
				}
			}
		}
	case "field_access":
		object := node.ChildByFieldName("object")
		field := node.ChildByFieldName("field")
		if object == nil || field == nil {
			return nil, false
		}
		owner, _ := staticFieldQualifierScope(object, source, ctx)
		resolution = findFieldResolutionInHierarchy(owner, field.Content(source), ctx)
	default:
		return nil, false
	}
	if resolution == nil || resolution.def.IsStatic || !volatileFieldDefinition(resolution.def) {
		return nil, false
	}
	return &volatileFieldAccess{resolution: resolution}, true
}

func volatileFieldStorageExpr(node *sitter.Node, access *volatileFieldAccess, source []byte, ctx Ctx) ast.Expr {
	if access.static != nil {
		return staticFieldStorageExpr(access.static, ctx)
	}
	node = unwrapParenthesizedExpressionNode(node)
	raw := ctx.Clone()
	// Suppress the read rewrite for precisely this storage root. Volatile fields
	// used inside its receiver are still ordinary reads and are not suppressed.
	raw.volatileStorageRoot = node
	return ParseExpr(node, source, raw)
}

func volatileStorageRoot(node *sitter.Node, ctx Ctx) bool {
	root := ctx.volatileStorageRoot
	return root != nil && node != nil && root.StartByte() == node.StartByte() && root.EndByte() == node.EndByte() && root.Type() == node.Type()
}

func volatileFieldValueType(access *volatileFieldAccess, ctx Ctx) ast.Expr {
	// Construct the declaration's physical type in the consuming package. A
	// reflection accessor is emitted in its owner, but a direct load can be in a
	// different package and must preserve its qualified Go/import representation.
	owner, field := access.resolution.owner, access.resolution.def
	javaType := qualifyJavaTypeInDeclaringContext(field.OriginalType, owner)
	value := abstractClassToInterface(javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx), javaType, ctx)
	return directOwnerTypeParameterFieldStorageType(owner, field, value, ctx)
}

func lowerVolatileFieldRead(node *sitter.Node, source []byte, ctx Ctx) (ast.Expr, bool) {
	if volatileStorageRoot(node, ctx) {
		return nil, false
	}
	access, ok := resolveVolatileFieldAccess(node, source, ctx)
	if !ok {
		return nil, false
	}
	valueType := volatileFieldValueType(access, ctx)
	value := volatileFieldLoad(volatileFieldStorageExpr(node, access, source, ctx), valueType, ctx, access.resolution.def, access.resolution.owner)
	if access.static == nil {
		return value, true
	}
	body := staticFieldPrelude(access.static, source, ctx, true)
	body = append(body, &ast.ReturnStmt{Results: []ast.Expr{value}})
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: valueType}}}}, Body: &ast.BlockStmt{List: body}}}, true
}

// A compound expression saves a volatile read, evaluates its RHS without any
// cell lock held, then performs a separate volatile write. It must never become
// CAS, Swap, GetAndAdd, or one lock enclosing the complete expression.
func lowerVolatileFieldAssignment(node *sitter.Node, source []byte, ctx Ctx) (ast.Expr, bool) {
	lhs, op, rhsNode, valid := assignmentExpressionNodes(node, source)
	if !valid {
		return nil, false
	}
	access, ok := resolveVolatileFieldAccess(lhs, source, ctx)
	if !ok {
		return nil, false
	}
	storage := volatileFieldStorageExpr(lhs, access, source, ctx)
	valueType := volatileFieldValueType(access, ctx)
	javaType := qualifyJavaTypeInDeclaringContext(access.resolution.def.OriginalType, access.resolution.owner)
	body := []ast.Stmt{}
	if access.static != nil {
		body = staticFieldPrelude(access.static, source, ctx, false)
	}
	receiverName := staticFieldTempName(node, source, ctx, "__java2goVolatileReceiver")
	if access.static == nil {
		selector, selectorOK := storage.(*ast.SelectorExpr)
		if !selectorOK {
			return &ast.BadExpr{}, true
		}
		// Save the receiver without dereferencing it. For a simple putfield the RHS
		// completes before the null receiver failure; a compound getfield fails first.
		body = append(body, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(receiverName)}, Tok: token.DEFINE, Rhs: []ast.Expr{selector.X}})
		storage = &ast.SelectorExpr{X: stdjavaCall(ctx, "ReferenceRequireNonNull", ast.NewIdent(receiverName)), Sel: selector.Sel}
	}
	operator := op.Content(source)
	valueName := staticFieldTempName(node, source, ctx, "__java2goVolatileValue", receiverName)
	if operator == "=" {
		rhsCtx := ctx.Clone()
		rhsCtx.expectedType = javaType
		rhsCtx.expectedTypeRoot = rhsNode
		rhs := coerceArgumentToExpectedType(ParseExpr(rhsNode, source, rhsCtx), rhsNode, javaType, ctx, source)
		body = append(body, typedLocalDeclaration(valueName, valueType, rhs))
		if access.static != nil {
			if ensure := staticFieldEnsureStmt(access.static, ctx); ensure != nil {
				body = append(body, ensure)
			}
		}
	} else {
		if access.static != nil {
			if ensure := staticFieldEnsureStmt(access.static, ctx); ensure != nil {
				body = append(body, ensure)
			}
		}
		oldName := staticFieldTempName(node, source, ctx, "__java2goVolatileOld", receiverName, valueName)
		oldValue := volatileFieldLoad(storage, valueType, ctx, access.resolution.def, access.resolution.owner)
		operationType := javaType
		oldType := valueType
		primitive, boxed := builtinJavaWrapperPrimitive(javaType, ctx)
		if boxed {
			operationType = primitive
			oldType = javaTypeStringToGoTypeExpr(primitive, inScopeTypeParameters(ctx), ctx)
			oldValue = javaUnboxExpr(oldValue, javaType, ctx)
		}
		body = append(body, typedLocalDeclaration(oldName, oldType, oldValue))
		rhsType, known := inferExprJavaType(rhsNode, ctx, source)
		if !known || strings.TrimSpace(rhsType) == "" {
			rhsType = "Object"
		}
		rhs := ParseExpr(rhsNode, source, ctx)
		if _, numeric := javaPrimitiveType(operationType); numeric {
			if rhsPrimitive, rhsBoxed := builtinJavaWrapperPrimitive(rhsType, ctx); rhsBoxed {
				rhs = javaUnboxExpr(rhs, rhsType, ctx)
				rhsType = rhsPrimitive
			}
		}
		rhsName := staticFieldTempName(node, source, ctx, "__java2goVolatileRHS", receiverName, oldName, valueName)
		body = append(body, typedLocalDeclaration(rhsName, javaTypeStringToGoTypeExpr(rhsType, inScopeTypeParameters(ctx), ctx), rhs))
		value, supported := compoundAssignmentValue(operator, ast.NewIdent(oldName), ast.NewIdent(rhsName), operationType, rhsType, ctx)
		if !supported {
			return &ast.BadExpr{}, true
		}
		if boxed {
			value = javaBoxExpr(value, primitive, ctx)
		}
		body = append(body, typedLocalDeclaration(valueName, valueType, value))
	}
	body = append(body, &ast.ExprStmt{X: stdjavaGenericCall(ctx, "VolatileStore", []ast.Expr{valueType}, []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: storage}, ast.NewIdent(valueName)})}, &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent(valueName)}})
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: valueType}}}}, Body: &ast.BlockStmt{List: body}}}, true
}

func lowerVolatileFieldUpdate(node, operand *sitter.Node, post, increment bool, source []byte, ctx Ctx) (ast.Expr, bool) {
	access, ok := resolveVolatileFieldAccess(operand, source, ctx)
	if !ok {
		return nil, false
	}
	storage := volatileFieldStorageExpr(operand, access, source, ctx)
	valueType := volatileFieldValueType(access, ctx)
	javaType := qualifyJavaTypeInDeclaringContext(access.resolution.def.OriginalType, access.resolution.owner)
	body := []ast.Stmt{}
	if access.static != nil {
		body = staticFieldPrelude(access.static, source, ctx, true)
	}
	if selector, ok := storage.(*ast.SelectorExpr); ok && access.static == nil {
		storage = &ast.SelectorExpr{X: stdjavaCall(ctx, "ReferenceRequireNonNull", selector.X), Sel: selector.Sel}
	}
	dstName := staticFieldTempName(node, source, ctx, "__java2goVolatileCell")
	oldName := staticFieldTempName(node, source, ctx, "__java2goVolatileOld", dstName)
	valueName := staticFieldTempName(node, source, ctx, "__java2goVolatileValue", dstName, oldName)
	body = append(body, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(dstName)}, Tok: token.DEFINE, Rhs: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: storage}}}, typedLocalDeclaration(oldName, valueType, stdjavaGenericCall(ctx, "VolatileLoad", []ast.Expr{valueType}, volatileLoadTypeArgument([]ast.Expr{ast.NewIdent(dstName)}, access.resolution.def, access.resolution.owner, ctx))))
	old := ast.Expr(ast.NewIdent(oldName))
	operationType := javaType
	primitive, boxed := builtinJavaWrapperPrimitive(javaType, ctx)
	if boxed {
		old = javaUnboxExpr(old, javaType, ctx)
		operationType = primitive
	}
	operator := "+="
	if !increment {
		operator = "-="
	}
	value, supported := compoundAssignmentValue(operator, old, &ast.BasicLit{Kind: token.INT, Value: "1"}, operationType, "int", ctx)
	if !supported {
		return &ast.BadExpr{}, true
	}
	if boxed {
		value = javaBoxExpr(value, primitive, ctx)
	}
	body = append(body, typedLocalDeclaration(valueName, valueType, value), &ast.ExprStmt{X: stdjavaGenericCall(ctx, "VolatileStore", []ast.Expr{valueType}, []ast.Expr{ast.NewIdent(dstName), ast.NewIdent(valueName)})})
	result := valueName
	if post {
		result = oldName
	}
	body = append(body, &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent(result)}})
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: valueType}}}}, Body: &ast.BlockStmt{List: body}}}, true
}

func volatileFieldAccessorName(scope *symbol.ClassScope, field *symbol.Definition) string {
	return symbol.GoIdentifier(collisionSafeExecutionIdentifier("Java2goVolatileFieldCell"+field.Name+"Java2goExecution", scope))
}

func volatileFieldAccessorDecls(scope *symbol.ClassScope, ctx Ctx) []ast.Decl {
	if scope == nil || scope.IsInterface {
		return nil
	}
	var result []ast.Decl
	for _, field := range scope.Fields {
		if field.IsStatic || !volatileFieldDefinition(field) || !sourceReflectionDeclaredField(scope, field) {
			continue
		}
		result = append(result, &ast.FuncDecl{Name: ast.NewIdent(volatileFieldAccessorName(scope, field)), Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("receiver")}, Type: classSubobjectPointerTypeExpr(scope, scope.GoTypeParameterNames(), scope, ctx)}}}, Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}, Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("VolatileFieldCell", ctx)}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: &ast.SelectorExpr{X: ast.NewIdent("receiver"), Sel: ast.NewIdent(symbol.GoIdentifier(field.Name))}}}}}}})
	}
	return result
}

func volatileFieldDescriptorCallback(scope *symbol.ClassScope, field *symbol.Definition, ctx Ctx) ast.Expr {
	params := []*ast.Field{executionParameterField("execution", ctx), {Names: []*ast.Ident{ast.NewIdent("receiver")}, Type: ast.NewIdent("any")}}
	var cell ast.Expr
	if field.IsStatic {
		cell = &ast.UnaryExpr{Op: token.AND, X: ast.NewIdent(symbol.GoIdentifier(field.Name))}
	} else {
		cell = stdjavaCall(ctx, "ReflectGeneratedVolatileFieldCellExecution", ast.NewIdent("execution"), ast.NewIdent("receiver"), javaTypeIDLiteral(sourceClassRuntimeTypeID(scope, ctx), ctx), metadataString(volatileFieldAccessorName(scope, field)))
	}
	return &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{List: params}, Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("VolatileFieldCell", ctx)}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{cell}}}}}
}
