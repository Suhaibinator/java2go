package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// A raw Function's slots have the implicit Object origin of the canonical
// declaration. They are not source-written Object names or caller binders.
// The body plan belongs to one lambda scope and never changes Java inference.
type rawFunctionTargetABI struct {
	file  *symbol.FileScope
	owner *symbol.ClassScope
	scope *symbol.Definition
	root  *sitter.Node
}

func rawCanonicalFunctionTarget(javaType string, ctx Ctx) *rawFunctionTargetABI {
	component, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	base, arguments, valid := splitJavaMemberType(component)
	if !valid || rank != 0 || len(arguments) != 0 {
		return nil
	}
	if visibleTypeParameterDeclarationForJavaType(base, ctx) != nil {
		return nil
	}
	parts := strings.Split(base, ".")
	for count := 1; count < len(parts); count++ {
		prefix := strings.Join(parts[:count], ".")
		if resolveClassScopeByQualifiedName(ctx, prefix) != nil || visibleTypeParameterDeclarationForJavaType(prefix, ctx) != nil {
			return nil
		}
	}
	if !isExternalFunctionType(javaType, ctx) {
		return nil
	}
	return &rawFunctionTargetABI{file: ctx.currentFile, owner: ctx.currentClass}
}

func rawCanonicalFunctionLambdaTarget(javaType string, ctx Ctx) *rawFunctionTargetABI {
	root := ctx.expectedTypeRoot
	for root != nil && root.Type() == "parenthesized_expression" && nodeutil.SemanticNamedChildCount(root) == 1 {
		root = nodeutil.SemanticNamedChild(root, 0)
	}
	if root == nil || root.Type() != "lambda_expression" {
		return nil
	}
	plan := rawCanonicalFunctionTarget(javaType, ctx)
	if plan == nil {
		return nil
	}
	parameters := root.ChildByFieldName("parameters")
	if parameters == nil {
		return nil
	}
	switch parameters.Type() {
	case "formal_parameters":
		if nodeutil.SemanticNamedChildCount(parameters) != 1 || ctx.currentFile == nil {
			return nil
		}
		parameter := nodeutil.SemanticNamedChild(parameters, 0)
		if parameter.Type() != "formal_parameter" {
			return nil
		}
		typeNode := parameter.ChildByFieldName("type")
		if typeNode == nil {
			return nil
		}
		written := typeNode.Content(ctx.currentFile.Source)
		if !overrideBridgeCanonicalObjectResult(written, written, ctx) {
			return nil
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(parameter) {
			if child.Type() == "dimensions" {
				return nil
			}
		}
	case "inferred_parameters":
		if nodeutil.SemanticNamedChildCount(parameters) != 1 {
			return nil
		}
	case "identifier":
	default:
		return nil
	}
	plan.root = root
	return plan
}

func (plan *rawFunctionTargetABI) body(ctx Ctx) *rawFunctionTargetABI {
	copy := *plan
	copy.scope = ctx.localScope
	return &copy
}

func (plan *rawFunctionTargetABI) active(ctx Ctx) bool {
	return plan != nil && plan.scope != nil && ctx.localScope == plan.scope &&
		ctx.currentFile == plan.file && ctx.currentClass == plan.owner
}

func (plan *rawFunctionTargetABI) resultBoundary(node *sitter.Node, ctx Ctx) bool {
	return plan.active(ctx) && ctx.expectedType == plan.scope.OriginalType &&
		expectedTypeTargetsExpression(ctx, node)
}

func (plan *rawFunctionTargetABI) result(value ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if isStaticallyNullReference(node) {
		return ast.NewIdent("nil")
	}
	if actual, known := inferExprJavaType(node, ctx, source); known {
		if primitive, ok := javaPrimitiveType(actual); ok {
			return javaBoxExpr(value, primitive, ctx)
		}
	}
	return value
}

func (plan *rawFunctionTargetABI) adapter(value ast.Expr, executionAware bool, ctx Ctx) ast.Expr {
	constructor := "NewPlainFunctionFuncAdapter"
	if executionAware {
		constructor = "NewFunctionFuncAdapter"
	}
	return stdjavaGenericCall(ctx, constructor, []ast.Expr{ast.NewIdent("any"), ast.NewIdent("any")}, []ast.Expr{value})
}
