package transpiler

import (
	"go/ast"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func canonicalReentrantLockOwner(base string, parameters []string, ctx Ctx) bool {
	first := strings.SplitN(strings.TrimSpace(base), ".", 2)[0]
	if visibleTypeParameterDeclarationForJavaType(first, ctx) != nil || containsString(parameters, first) || resolveClassScopeByQualifiedName(ctx, first) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return false
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	return known && owner == "java.util.concurrent.locks.ReentrantLock" && intrinsicOwnerSupported(owner)
}

// Array lowering retains its dimensions outside this component-type hook.
func reentrantLockRuntimeTypeExpr(javaType string, typeArgs, typeParams []string, ctx Ctx) (ast.Expr, bool) {
	element, _ := javaArrayTypeParts(strings.TrimSpace(javaType))
	base, arguments := parseJavaTypeString(element)
	if !canonicalReentrantLockOwner(base, typeParams, ctx) {
		return nil, false
	}
	if len(typeArgs) != 0 || len(arguments) != 0 {
		return &ast.BadExpr{}, true
	}
	return &ast.StarExpr{X: stdjavaQualifiedExpr("JavaReentrantLock", ctx)}, true
}

func reentrantLockReferenceDeclaringContext(origin inferredJavaTypeOrigin, ctx Ctx) Ctx {
	if origin.declaringOwner == nil {
		return ctx
	}
	proof := ctx.Clone()
	proof.currentFile = nil
	proof.localScope = nil
	proof.localClasses = nil
	proof.localBindingBody = nil
	proof.syntheticTypeParameters = nil
	return classScopeCtx(origin.declaringOwner, proof)
}

func reentrantLockCanonicalReceiverApplicable(object *sitter.Node, ctx Ctx, source []byte) bool {
	if object == nil || object.Type() == "super" {
		return false
	}
	var origin inferredJavaTypeOrigin
	javaType, known := inferExprJavaType(object, ctx, source, &origin)
	if !known || origin.unresolved || origin.parameter != nil || origin.nominalScope != nil {
		return false
	}
	element, rank := javaArrayTypeParts(javaType)
	base, arguments := parseJavaTypeString(element)
	return rank == 0 && len(arguments) == 0 && canonicalReentrantLockOwner(base, nil, reentrantLockReferenceDeclaringContext(origin, ctx))
}

func reentrantLockConstructorApplicable(node *sitter.Node, typeArgs, args []ast.Expr, ctx Ctx, source []byte) bool {
	if node == nil || len(typeArgs) != 0 || len(args) != 0 || objectCreationClassBody(node) != nil {
		return false
	}
	typeNode := node.ChildByFieldName("type")
	if typeNode == nil {
		return false
	}
	base, arguments := parseJavaTypeString(typeNode.Content(source))
	return len(arguments) == 0 && canonicalReentrantLockOwner(base, nil, ctx)
}

func init() {
	registerIntrinsicOwner("java.util.concurrent.locks.ReentrantLock", true)
	registerConstructorNodeIntrinsic("ReentrantLock", func(typeArgs, args []ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if !reentrantLockConstructorApplicable(node, typeArgs, args, ctx, source) {
			return unsupportedIntrinsicValue(node, "java.util.concurrent.locks.ReentrantLock", source, ctx)
		}
		return stdjavaCall(ctx, "NewReentrantLock")
	})
	for _, specification := range []struct{ method, runtime, result string }{
		{"lock", "ReentrantLockLockExecution", "void"},
		{"unlock", "ReentrantLockUnlockExecution", "void"},
		{"getHoldCount", "ReentrantLockGetHoldCountExecution", "int"},
		{"isHeldByCurrentThread", "ReentrantLockIsHeldByCurrentThreadExecution", "boolean"},
		{"isLocked", "ReentrantLockIsLocked", "boolean"},
	} {
		registerInstanceIntrinsicResultType("ReentrantLock", specification.method, specification.result)
		registerInstanceNodeIntrinsic("ReentrantLock", specification.method, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if invocationArgumentCount(node) != 0 || !reentrantLockCanonicalReceiverApplicable(node.ChildByFieldName("object"), ctx, source) {
				return unsupportedIntrinsicValue(node, specification.result, source, ctx)
			}
			if specification.method == "isLocked" {
				return stdjavaCall(ctx, specification.runtime, receiver)
			}
			if executionExpr(ctx) == nil {
				return unsupportedIntrinsicValue(node, specification.result, source, ctx)
			}
			return stdjavaCall(ctx, specification.runtime, executionExpr(ctx), receiver)
		})
	}
}
