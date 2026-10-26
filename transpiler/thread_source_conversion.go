package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

func threadSourceTarget(javaType string, def *symbol.Definition, declaring Ctx) (ast.Expr, bool) {
	owner := threadSourceCanonicalOwner(javaType, def, declaring)
	switch owner {
	case "java.lang.Thread":
		return &ast.StarExpr{X: stdjavaQualifiedExpr("Thread", declaring)}, true
	case "java.lang.Runnable":
		return stdjavaQualifiedExpr("Runnable", declaring), true
	}
	return nil, false
}
func threadSourceOperand(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil {
		return false
	}
	inference := ctx.Clone()
	inference.expectedType = ""
	inference.expectedTypeRoot = nil
	var origin inferredJavaTypeOrigin
	typ, known := inferExprJavaType(node, inference, source, &origin)
	_, rank := javaArrayTypeParts(typ)
	return known && rank == 0 && !origin.unresolved && origin.parameter == nil && origin.nominalScope != nil && threadSourceRoot(origin.nominalScope, ctx) != nil
}
func threadSourceProjection(value ast.Expr, ctx Ctx) ast.Expr {
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{&ast.StarExpr{X: stdjavaQualifiedExpr("Thread", ctx)}}, []ast.Expr{value, stdjavaQualifiedExpr("ThreadTypeID", ctx)})
}
func threadSourceBoundaryConversion(value ast.Expr, node *sitter.Node, expected string, ctx Ctx, source []byte) (ast.Expr, bool) {
	if _, ok := threadSourceBoundaryTarget(node, expected, ctx, source); !ok || !threadSourceOperand(node, ctx, source) {
		return nil, false
	}
	return threadSourceProjection(value, ctx), true
}
func threadSourceFormalTarget(resolution *methodResolution, index int, ctx Ctx) (ast.Expr, bool) {
	if resolution == nil || resolution.def == nil || index < 0 || index >= len(resolution.def.Parameters) || executionParameterIsVariadic(resolution.def, index) {
		return nil, false
	}
	declaring, ok := threadSourceDeclarationCtx(resolution.owner, resolution.def, ctx)
	if !ok {
		return nil, false
	}
	p := resolution.def.Parameters[index]
	if p == nil {
		return nil, false
	}
	return threadSourceTarget(p.OriginalType, p, declaring)
}
func threadSourceInvocationArguments(args []ast.Expr, node *sitter.Node, resolution *methodResolution, ctx Ctx, source []byte) {
	nodes := nodeutil.SemanticNamedChildrenOf(node)
	for i, value := range args {
		if i >= len(nodes) {
			break
		}
		if _, ok := threadSourceFormalTarget(resolution, i, ctx); ok && threadSourceOperand(nodes[i], ctx, source) {
			args[i] = threadSourceProjection(value, ctx)
		}
	}
}
func threadSourceEquality(left, right *sitter.Node, ctx Ctx, source []byte) bool {
	eligible := func(node *sitter.Node) bool {
		if threadSourceOperand(node, ctx, source) {
			return true
		}
		if node == nil {
			return false
		}
		if node.Type() == "null_literal" {
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
			declaring, owned = threadSourceDeclarationCtx(origin.declaringOwner, nil, ctx)
			if !owned {
				return false
			}
		}
		_, ok = threadSourceTarget(typ, nil, declaring)
		return ok
	}
	return (threadSourceOperand(left, ctx, source) || threadSourceOperand(right, ctx, source)) && eligible(left) && eligible(right)
}
func threadSourceBoundaryTarget(node *sitter.Node, expected string, ctx Ctx, source []byte) (ast.Expr, bool) {
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
		declaring, ok := threadSourceDeclarationCtx(ctx.currentClass, ctx.localScope, ctx)
		if !ok {
			return nil, false
		}
		return threadSourceTarget(expected, ctx.localScope, declaring)
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
			return threadSourceTarget(expected, nil, ctx)
		}
		if declaration.Type() == "field_declaration" {
			name := parent.ChildByFieldName("name")
			if name == nil || ctx.currentClass == nil {
				return nil, false
			}
			field := ctx.currentClass.FindFieldByName(name.Content(source))
			declaring, ok := threadSourceDeclarationCtx(ctx.currentClass, nil, ctx)
			if field != nil && ok {
				return threadSourceTarget(expected, field, declaring)
			}
		}
	case "assignment_expression":
		if !sameSourceNode(parent.ChildByFieldName("right"), boundary) {
			return nil, false
		}
		left := parent.ChildByFieldName("left")
		if target, ok := threadSourceAssignmentTarget(left, expected, ctx, source); ok {
			return target, true
		}
		if selected, ok := resolveStaticFieldAccess(left, source, ctx); ok && selected.resolution != nil {
			field := selected.resolution
			declaring, ok := threadSourceDeclarationCtx(field.owner, nil, ctx)
			if ok && strings.TrimSpace(field.def.OriginalType) == expected {
				return threadSourceTarget(expected, field.def, declaring)
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
					declaring, ok := threadSourceDeclarationCtx(field.owner, nil, ctx)
					if ok {
						return threadSourceTarget(expected, field.def, declaring)
					}
				}
			}
		}
	}
	return nil, false
}

func threadSourceAssignmentTarget(left *sitter.Node, expected string, ctx Ctx, source []byte) (ast.Expr, bool) {
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
						return threadSourceTarget(expected, nil, ctx)
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
			declaring, ok := threadSourceDeclarationCtx(ctx.currentClass, ctx.localScope, ctx)
			if ok {
				return threadSourceTarget(expected, parameter, declaring)
			}
			return nil, false
		}
	}
	field := findFieldResolutionInHierarchy(ctx.currentClass, name, ctx)
	if field != nil && strings.TrimSpace(field.def.OriginalType) == expected {
		declaring, ok := threadSourceDeclarationCtx(field.owner, nil, ctx)
		if ok {
			return threadSourceTarget(expected, field.def, declaring)
		}
	}
	return nil, false
}

// The Thread declaration contributes Runnable independently of any source type
// arguments. Native endpoints are resolved before following donor ancestry.
func threadSourceRuntimeAssignable(actual *symbol.ClassScope, arguments []string, expected string, ctx Ctx) bool {
	if actual == nil || actual.Class == nil || (len(arguments) != 0 && len(arguments) != len(actual.TypeParameters)) {
		return false
	}
	owner := threadSourceCanonicalOwner(expected, nil, ctx)
	return (owner == "java.lang.Thread" || owner == "java.lang.Runnable") && threadSourceRoot(actual, ctx) != nil
}

// These are retained canonical declaration endpoints, never caller spellings.
func threadSourceCanonicalOriginAssignable(actual, formal inferredJavaTypeOrigin) bool {
	closed := func(origin inferredJavaTypeOrigin) bool {
		return completeJavaTypeOrigin(origin) && !origin.unresolved && origin.parameter == nil && origin.nominalScope == nil && origin.arrayRank == 0 && !origin.wildcard && len(origin.arguments) == 0
	}
	return closed(actual) && closed(formal) && actual.externalName == "java.lang.Thread" && formal.externalName == "java.lang.Runnable"
}
func threadSourceNativeReferenceAssignable(actual, expected string, ctx Ctx) bool {
	a := threadSourceCanonicalOwner(actual, nil, ctx)
	f := threadSourceCanonicalOwner(expected, nil, ctx)
	if a == "" || f == "" {
		return false
	}
	return threadSourceCanonicalOriginAssignable(intrinsicJavaTypeOrigin(a), intrinsicJavaTypeOrigin(f))
}
