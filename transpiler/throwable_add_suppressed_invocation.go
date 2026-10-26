package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

// Resolve the inherited final(Throwable) declaration against source overloads
// before entering the Java synchronized body. Ordinary source owners retain
// their declarations; String/Object overloads are selected by argument type.
func throwableAddSuppressedInvocation(object *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if object == nil || invocationArgumentCount(object.Parent()) != 1 {
		return nil
	}
	declaring := ctx
	var receiverType string
	var known bool
	if object.Type() == "super" {
		if ctx.currentClass == nil {
			return nil
		}
		declaring = classScopeCtx(ctx.currentClass, ctx)
		receiverType, known = ctx.currentClass.Superclass, true
	} else {
		receiverType, known = inferExprJavaType(object, ctx, source)
	}
	if !known || !isExceptionJavaType(declaring, receiverType) {
		return nil
	}
	argument := invocationArgumentNode(object.Parent(), 0)
	if argument == nil {
		return nil
	}
	argumentType, argumentKnown := inferExprJavaType(argument, ctx, source)
	_, rank := javaArrayTypeParts(strings.TrimSpace(argumentType))
	if argument.Type() != "null_literal" && (!argumentKnown || rank != 0 || !javaExceptionReferenceAssignable(argumentType, "java.lang.Throwable", ctx)) {
		return nil
	}
	if target := resolveInvocationTarget(object, declaring, source); target != nil {
		selected, _ := findBestMethodForInvocationTarget(target, "addSuppressed", object.Parent().ChildByFieldName("arguments"), true, true, declaring, source)
		if selected != nil && selected.def != nil {
			// Fixed arity inherited(Throwable) precedes source variable-arity packing.
			count := len(selected.def.Parameters)
			packed := count != 0 && executionParameterIsVariadic(selected.def, count-1) && !selected.expandVarargsArray
			if !packed {
				if count != 1 {
					return nil
				}
				formalCtx := classScopeCtx(selected.owner, ctx)
				formal := qualifyDeclaredReferenceType(definitionDeclaredJavaType(selected.def.Parameters[0]), formalCtx)
				if !javaInferenceTypeAssignable("java.lang.Throwable", formal, formalCtx) {
					return nil
				}
			}
		}
	}
	var receiver ast.Expr
	if object.Type() == "super" {
		receiver = ast.NewIdent(sourceInstanceReceiverName(ctx))
	} else {
		receiver = ParseExpr(object, source, ctx)
	}
	// Capture the receiver read before an argument can reassign its source local.
	receiver = snapshotJavaExpressionValue(receiver, ctx)
	return stdjavaCall(ctx, "JavaThrowableAddSuppressedExecution", intrinsicExecutionExpr(ctx), receiver, ParseExpr(argument, source, ctx))
}
