package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

// Source declarations and actual binders remain source dispatch, even where
// erasure happens to be Thread. Declaration origin precedes caller namespace.
func threadNativeStateReceiverApplicable(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil || invocationArgumentCount(node) != 0 {
		return false
	}
	object := node.ChildByFieldName("object")
	if object == nil {
		return false
	}
	var origin inferredJavaTypeOrigin
	actual, known := inferExprJavaType(object, ctx, source, &origin)
	if !known || origin.unresolved || origin.parameter != nil || origin.nominalScope != nil {
		return false
	}
	proof := ctx
	if origin.declaringOwner != nil {
		proof = ctx.Clone()
		proof.currentFile = nil
		proof.syntheticTypeParameters = nil
		proof = classScopeCtx(origin.declaringOwner, proof)
		if proof.currentFile == nil {
			return false
		}
		actual = origin.javaType
	} else if origin.javaType != "" {
		actual = origin.javaType
	}
	element, rank := javaArrayTypeParts(strings.TrimSpace(actual))
	base, arguments := parseJavaTypeString(element)
	if rank != 0 || len(arguments) != 0 || visibleTypeParameterDeclarationForJavaType(base, proof) != nil || resolveClassScopeByQualifiedName(proof, base) != nil {
		return false
	}
	owner, canonical := canonicalIntrinsicOwner(base, proof)
	return canonical && owner == "java.lang.Thread"
}

func threadNativeStateStaticApplicable(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil || invocationArgumentCount(node) != 0 {
		return false
	}
	object := node.ChildByFieldName("object")
	if object == nil {
		return resolveStaticImportedMethod(node, ctx, source).intrinsic == "Thread"
	}
	owner, canonical := intrinsicStaticClassName(object, ctx, source)
	return canonical && owner == "Thread"
}

func threadNativeStateResult(method string) string {
	if method == "onSpinWait" {
		return "void"
	}
	return "boolean"
}

func registerThreadNativeStateIntrinsics() {
	registerIntrinsicOwner("java.lang.Thread", true)
	for _, method := range []string{"isInterrupted", "interrupted", "onSpinWait"} {
		method := method
		registerInstanceNodeIntrinsic("Thread", method, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !threadNativeStateReceiverApplicable(node, ctx, source) || executionExpr(ctx) == nil {
				return unsupportedIntrinsicValue(node, threadNativeStateResult(method), source, ctx)
			}
			execution := executionExpr(ctx)
			if method == "isInterrupted" {
				return stdjavaCall(ctx, "ThreadIsInterruptedExecution", execution, receiver)
			}
			var call ast.Expr
			var results *ast.FieldList
			if method == "interrupted" {
				call = stdjavaCall(ctx, "ThreadInterruptedExecution", execution)
				results = &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}}
			} else {
				call = stdjavaCall(ctx, "ThreadOnSpinWaitExecution", execution)
			}
			return stageStaticQualifierWithResults(receiver, call, results)
		})
		registerInstanceIntrinsicDerivedResultType("Thread", method, func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
			if threadNativeStateReceiverApplicable(node, ctx, source) {
				return threadNativeStateResult(method), true
			}
			return "", false
		})
		// Register the prohibited instance-as-static signature to diagnose it
		// explicitly; only the two true static declarations have result metadata.
		registerStaticNodeIntrinsic("Thread", method, func(node *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
			if method == "isInterrupted" || !threadNativeStateStaticApplicable(node, ctx, source) || executionExpr(ctx) == nil {
				return unsupportedIntrinsicValue(node, threadNativeStateResult(method), source, ctx)
			}
			if method == "interrupted" {
				return stdjavaCall(ctx, "ThreadInterruptedExecution", executionExpr(ctx))
			}
			return stdjavaCall(ctx, "ThreadOnSpinWaitExecution", executionExpr(ctx))
		})
		if method != "isInterrupted" {
			registerStaticIntrinsicImportSignature("Thread", method)
			registerStaticIntrinsicDerivedResultType("Thread", method, func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
				if threadNativeStateStaticApplicable(node, ctx, source) {
					return threadNativeStateResult(method), true
				}
				return "", false
			})
		}
	}
}
