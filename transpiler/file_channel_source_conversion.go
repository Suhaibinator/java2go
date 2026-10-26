package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/token"
	"strings"
)

// A signature is interpreted in the declaring file and binder scope. The
// consumer's same-spelled class, local class, or type parameter cannot own it.
func fileChannelSourceDeclarationCtx(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) (Ctx, bool) {
	if owner == nil || findFileScopeForClassScope(owner, ctx) == nil {
		return ctx, false
	}
	declaring := ctx.Clone()
	declaring.currentFile = nil
	declaring.syntheticTypeParameters = nil
	declaring.localClasses = nil
	declaring = classScopeCtx(owner, declaring)
	declaring.localScope = method
	return declaring, declaring.currentFile != nil
}
func fileChannelSourceTarget(javaType string, def *symbol.Definition, declaring Ctx) (ast.Expr, bool) {
	base, _ := parseJavaTypeString(strings.TrimSpace(javaType))
	if def != nil && (def.DirectTypeParameter != nil || def.TypeParameterBindings[base] != nil) {
		return nil, false
	}
	owner, ok := fileChannelDeclarationOwner(javaType, declaring)
	if !ok {
		return nil, false
	}
	switch owner {
	case "java.nio.channels.FileChannel", "java.nio.channels.ReadableByteChannel", "java.nio.channels.WritableByteChannel":
		return fileChannelDeclarationRuntimeTypeExpr(javaType, declaring)
	}
	return nil, false
}
func fileChannelSourceOperand(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil {
		return false
	}
	inference := ctx.Clone()
	inference.expectedType = ""
	inference.expectedTypeRoot = nil
	var origin inferredJavaTypeOrigin
	_, known := inferExprJavaType(node, inference, source, &origin)
	return known && !origin.unresolved && origin.parameter == nil && origin.nominalScope != nil && fileChannelSourceRoot(origin.nominalScope, ctx) != nil
}

// Projection evaluates once and returns a genuine nil for a null source view.
// Both the source and native view retain their allocation's shared ObjectInfo.
func fileChannelSourceProjection(value ast.Expr, target ast.Expr, ctx Ctx) ast.Expr {
	local := ast.NewIdent("__java2goChannelSource")
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: target}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{
		&ast.AssignStmt{Lhs: []ast.Expr{local}, Tok: token.DEFINE, Rhs: []ast.Expr{value}},
		&ast.IfStmt{Cond: stdjavaCall(ctx, "JavaReferenceEqual", local, ast.NewIdent("nil")), Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("nil")}}}}},
		&ast.ReturnStmt{Results: []ast.Expr{methodCall(local, fileChannelSourceViewMethod)}},
	}}}}
}
func fileChannelSourceBoundaryTarget(node *sitter.Node, expected string, ctx Ctx, source []byte) (ast.Expr, bool) {
	if node == nil {
		return nil, false
	}
	boundary := node
	for boundary.Parent() != nil && boundary.Parent().Type() == "parenthesized_expression" {
		boundary = boundary.Parent()
	}
	parent := boundary.Parent()
	if parent == nil {
		return nil, false
	}
	switch parent.Type() {
	case "return_statement":
		if ctx.localScope == nil || !receiverNodeWithin(node, ctx.localScope.DeclarationNode) || strings.TrimSpace(ctx.localScope.OriginalType) != expected {
			return nil, false
		}
		for n := boundary.Parent(); n != nil && !sameSourceNode(n, ctx.localScope.DeclarationNode); n = n.Parent() {
			if n.Type() == "lambda_expression" || n.Type() == "class_declaration" {
				return nil, false
			}
		}
		declaring, ok := fileChannelSourceDeclarationCtx(ctx.currentClass, ctx.localScope, ctx)
		if !ok {
			return nil, false
		}
		return fileChannelSourceTarget(expected, ctx.localScope, declaring)
	case "variable_declarator":
		if !sameSourceNode(parent.ChildByFieldName("value"), boundary) {
			return nil, false
		}
		declaration := parent.Parent()
		if declaration == nil {
			return nil, false
		}
		typ := declaration.ChildByFieldName("type")
		if typ == nil || strings.TrimSpace(typ.Content(source)) != expected {
			return nil, false
		}
		if declaration.Type() == "local_variable_declaration" {
			return fileChannelSourceTarget(expected, nil, ctx)
		}
		if declaration.Type() == "field_declaration" {
			name := parent.ChildByFieldName("name")
			if name == nil || ctx.currentClass == nil {
				return nil, false
			}
			field := ctx.currentClass.FindFieldByName(name.Content(source))
			declaring, ok := fileChannelSourceDeclarationCtx(ctx.currentClass, nil, ctx)
			if field != nil && ok {
				return fileChannelSourceTarget(expected, field, declaring)
			}
		}
	case "assignment_expression":
		if !sameSourceNode(parent.ChildByFieldName("right"), boundary) {
			return nil, false
		}
		left := parent.ChildByFieldName("left")
		if target, ok := fileChannelSourceAssignmentTarget(left, expected, ctx, source); ok {
			return target, true
		}
		if selected, ok := resolveStaticFieldAccess(left, source, ctx); ok && selected.resolution != nil {
			field := selected.resolution
			declaring, ok := fileChannelSourceDeclarationCtx(field.owner, nil, ctx)
			if ok && strings.TrimSpace(field.def.OriginalType) == expected {
				return fileChannelSourceTarget(expected, field.def, declaring)
			}
		}
		if left != nil && left.Type() == "field_access" {
			object := left.ChildByFieldName("object")
			var origin inferredJavaTypeOrigin
			_, known := inferExprJavaType(object, ctx, source, &origin)
			name := left.ChildByFieldName("field")
			if known && !origin.unresolved && origin.parameter == nil && origin.nominalScope != nil && name != nil {
				field := findFieldResolutionInHierarchy(origin.nominalScope, name.Content(source), ctx)
				if field != nil && strings.TrimSpace(field.def.OriginalType) == expected {
					declaring, ok := fileChannelSourceDeclarationCtx(field.owner, nil, ctx)
					if ok {
						return fileChannelSourceTarget(expected, field.def, declaring)
					}
				}
			}
		}
	}
	return nil, false
}
func fileChannelSourceBoundaryConversion(value ast.Expr, node *sitter.Node, expected string, ctx Ctx, source []byte) (ast.Expr, bool) {
	target, ok := fileChannelSourceBoundaryTarget(node, expected, ctx, source)
	if !ok || !fileChannelSourceOperand(node, ctx, source) {
		return nil, false
	}
	return fileChannelSourceProjection(value, target, ctx), true
}
func fileChannelSourceFormalTarget(resolution *methodResolution, index int, ctx Ctx) (ast.Expr, bool) {
	if resolution == nil || resolution.def == nil || index < 0 || index >= len(resolution.def.Parameters) || len(resolution.def.TypeParameters) != 0 || executionParameterIsVariadic(resolution.def, index) {
		return nil, false
	}
	declaring, ok := fileChannelSourceDeclarationCtx(resolution.owner, resolution.def, ctx)
	if !ok {
		return nil, false
	}
	parameter := resolution.def.Parameters[index]
	if parameter == nil {
		return nil, false
	}
	return fileChannelSourceTarget(parameter.OriginalType, parameter, declaring)
}
func fileChannelSourceInvocationArguments(args []ast.Expr, node *sitter.Node, resolution *methodResolution, ctx Ctx, source []byte) {
	nodes := nodeutil.SemanticNamedChildrenOf(node)
	for index, value := range args {
		if index >= len(nodes) {
			break
		}
		target, ok := fileChannelSourceFormalTarget(resolution, index, ctx)
		if !ok {
			continue
		}
		if nodes[index].Type() == "null_literal" {
			args[index] = ast.NewIdent("nil")
		} else if fileChannelSourceOperand(nodes[index], ctx, source) {
			args[index] = fileChannelSourceProjection(value, target, ctx)
		}
	}
}
func fileChannelSourceEquality(left, right *sitter.Node, ctx Ctx, source []byte) bool {
	eligible := func(node *sitter.Node) bool {
		if fileChannelSourceOperand(node, ctx, source) {
			return true
		}
		var origin inferredJavaTypeOrigin
		typ, ok := inferExprJavaType(node, ctx, source, &origin)
		if !ok || origin.unresolved || origin.parameter != nil || origin.nominalScope != nil {
			return false
		}
		declaring := ctx
		if origin.declaringOwner != nil {
			var owned bool
			declaring, owned = fileChannelSourceDeclarationCtx(origin.declaringOwner, nil, ctx)
			if !owned {
				return false
			}
		}
		_, ok = fileChannelSourceTarget(typ, nil, declaring)
		return ok
	}
	return (fileChannelSourceOperand(left, ctx, source) || fileChannelSourceOperand(right, ctx, source)) && eligible(left) && eligible(right)
}

// Inspect the lexical declarator that actually binds this assignment. The
// method-wide variable inventory cannot prove declaration order or shadowing.
func fileChannelSourceAssignmentTarget(left *sitter.Node, expected string, ctx Ctx, source []byte) (ast.Expr, bool) {
	if left == nil || left.Type() != "identifier" {
		return nil, false
	}
	name := left.Content(source)
	for child, parent := left, left.Parent(); parent != nil; child, parent = parent, parent.Parent() {
		if parent.Type() == "lambda_expression" {
			return nil, false
		}
		if parent.Type() == "block" || parent.Type() == "constructor_body" {
			for _, sibling := range nodeutil.SemanticNamedChildrenOf(parent) {
				if sibling.StartByte() >= child.StartByte() {
					break
				}
				if sibling.Type() != "local_variable_declaration" {
					continue
				}
				for _, declarator := range nodeutil.SemanticNamedChildrenOf(sibling) {
					ident := declarator.ChildByFieldName("name")
					if declarator.Type() == "variable_declarator" && ident != nil && ident.Content(source) == name {
						typ := sibling.ChildByFieldName("type")
						if typ == nil || strings.TrimSpace(typ.Content(source)) != expected {
							return nil, false
						}
						return fileChannelSourceTarget(expected, nil, ctx)
					}
				}
			}
		}
		if ctx.localScope != nil && sameSourceNode(parent, ctx.localScope.DeclarationNode) {
			break
		}
	}
	if ctx.localScope != nil {
		if parameter := ctx.localScope.ParameterByName(name); parameter != nil {
			if strings.TrimSpace(parameter.OriginalType) != expected {
				return nil, false
			}
			declaring, ok := fileChannelSourceDeclarationCtx(ctx.currentClass, ctx.localScope, ctx)
			if ok {
				return fileChannelSourceTarget(expected, parameter, declaring)
			}
			return nil, false
		}
	}
	field := findFieldResolutionInHierarchy(ctx.currentClass, name, ctx)
	if field != nil && strings.TrimSpace(field.def.OriginalType) == expected {
		declaring, ok := fileChannelSourceDeclarationCtx(field.owner, nil, ctx)
		if ok {
			return fileChannelSourceTarget(expected, field.def, declaring)
		}
	}
	return nil, false
}
