package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

// Only a receiver whose declared type is the canonical native Thread can select
// this zero-argument void declaration. Source declarations, binders and source
// subclasses retain their own dispatch; source-to-native carrier wiring remains
// a separate prerequisite.
func threadInterruptInvocationApplicable(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil || invocationArgumentCount(node) != 0 {
		return false
	}
	receiver := node.ChildByFieldName("object")
	actual, known := inferExprJavaType(receiver, ctx, source)
	if !known {
		return false
	}
	element, rank := javaArrayTypeParts(actual)
	base, args := parseJavaTypeString(strings.TrimSpace(element))
	if rank != 0 || len(args) != 0 || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return false
	}
	owner, canonical := canonicalIntrinsicOwner(base, ctx)
	return canonical && owner == "java.lang.Thread"
}

func registerThreadInterruptIntrinsics() {
	registerIntrinsicOwner("java.lang.Thread", true)
	registerInstanceNodeIntrinsic("Thread", "interrupt", func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if !threadInterruptInvocationApplicable(node, ctx, source) {
			return unsupportedIntrinsicValue(node, "void", source, ctx)
		}
		return stdjavaCall(ctx, "ThreadInterruptExecution", intrinsicExecutionExpr(ctx), receiver)
	})
	registerInstanceIntrinsicDerivedResultType("Thread", "interrupt", func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		if threadInterruptInvocationApplicable(node, ctx, source) {
			return "void", true
		}
		return "", false
	})
}
