package transpiler

import (
	"go/ast"

	sitter "github.com/smacker/go-tree-sitter"
)

// Considered after source member lookup, before lexical outer static lookup.
// A zero-argument builtin is applicable in the fixed-arity phase, before any
// source overload that requires packing an empty varargs array.
func inheritedBuiltinMessageSelected(node *sitter.Node, selected *methodResolution, ctx Ctx, source []byte) bool {
	if !builtinMessageWinsSourceResolution(selected) || node == nil ||
		node.ChildByFieldName("name").Content(source) != "getMessage" || invocationArgumentCount(node) != 0 {
		return false
	}
	if object := node.ChildByFieldName("object"); object != nil {
		return builtinMessageReceiver(object, ctx, source)
	}
	return ctx.localScope != nil && !ctx.localScope.IsStatic && sourceInheritsThrowable(ctx.currentClass, ctx)
}

func builtinMessageReceiver(object *sitter.Node, ctx Ctx, source []byte) bool {
	javaType, known := inferExprJavaType(object, ctx, source)
	if object.Type() == "super" && ctx.currentClass != nil {
		// super is a receiver-only expression, not an ordinary value with an
		// inferred type. Resolve its declared parent before choosing its body.
		javaType, known = ctx.currentClass.Superclass, true
	}
	return known && throwableMessageReceiverType(javaType, ctx)
}

func throwableMessageInvocation(object *sitter.Node, method string, ctx Ctx, source []byte) ast.Expr {
	if method != "getMessage" || object == nil || invocationArgumentCount(object.Parent()) != 0 {
		return nil
	}
	if !builtinMessageReceiver(object, ctx, source) {
		return nil
	}
	if target := resolveInvocationTarget(object, ctx, source); target != nil {
		if selected, _ := findBestMethodForInvocationTarget(target, method, object.Parent().ChildByFieldName("arguments"), true, false, ctx, source); !builtinMessageWinsSourceResolution(selected) {
			return nil
		}
	}
	if object.Type() == "super" {
		return stdjavaCall(ctx, "ThrowableMessageDefault", ast.NewIdent(ShortName(ctx.className)))
	}
	return stdjavaCall(ctx, "ThrowableMessageExecution", intrinsicExecutionExpr(ctx), ParseExpr(object, source, ctx))
}

// This decision is used only for a zero-argument getMessage invocation. An
// applicable source declaration with nonzero formal arity is necessarily a
// variable-arity invocation; the inherited zero-formal declaration wins first.
// A real zero-formal source override retains ordinary source dispatch.
func builtinMessageWinsSourceResolution(selected *methodResolution) bool {
	return selected == nil || selected.def == nil || len(selected.def.Parameters) != 0
}
