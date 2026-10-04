package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

func randomAccessFileStringConstructorApplicable(node *sitter.Node, typeArgs []ast.Expr, ctx Ctx, source []byte) bool {
	if node == nil || len(typeArgs) != 0 || invocationArgumentCount(node) != 2 {
		return false
	}
	typeNode := node.ChildByFieldName("type")
	if typeNode == nil {
		return false
	}
	element, rank := javaArrayTypeParts(typeNode.Content(source))
	base, args := parseJavaTypeString(strings.TrimSpace(element))
	if rank != 0 || len(args) != 0 || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return false
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	if !known || owner != "java.io.RandomAccessFile" {
		return false
	}
	path := unwrapParenthesizedExpressionNode(invocationArgumentNode(node, 0))
	// A bare null path has both the String and File overloads. Do not guess.
	if path == nil || path.Type() == "null_literal" {
		return false
	}
	return intrinsicInvocationConversionApplicable(invocationArgumentNode(node, 0), "java.lang.String", ctx, source) && intrinsicInvocationConversionApplicable(invocationArgumentNode(node, 1), "java.lang.String", ctx, source)
}
func registerRandomAccessFileReferenceIntrinsic() {
	registerIntrinsicOwner("java.io.RandomAccessFile", true)
	registerConstructorNodeIntrinsic("RandomAccessFile", func(typeArgs, args []ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if len(args) != 2 || !randomAccessFileStringConstructorApplicable(node, typeArgs, ctx, source) {
			return unsupportedIntrinsicValue(node, "java.io.RandomAccessFile", source, ctx)
		}
		path := coerceArgumentToExpectedType(args[0], invocationArgumentNode(node, 0), "java.lang.String", ctx, source)
		mode := coerceArgumentToExpectedType(args[1], invocationArgumentNode(node, 1), "java.lang.String", ctx, source)
		return stdjavaCall(ctx, "NewRandomAccessFileStringExecution", intrinsicExecutionExpr(ctx), path, mode)
	})
}
