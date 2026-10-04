package transpiler

import (
	"go/ast"

	sitter "github.com/smacker/go-tree-sitter"
)

// An explicit super call selects Object's empty body only when no applicable
// source zero-formal declaration intervenes. Object's inherited fixed-arity
// method also wins before a source overload that packs an empty varargs array.
func objectSuperFinalizeInvocation(object *sitter.Node, method string, ctx Ctx, source []byte) ast.Expr {
	if object == nil || object.Type() != "super" || method != "finalize" ||
		invocationArgumentCount(object.Parent()) != 0 || ctx.currentClass == nil ||
		ctx.currentClass.IsInterface || ctx.currentClass.IsEnum {
		return nil
	}
	// This shared ancestry proof resolves every extends clause in its declaring
	// context and admits only source chains terminating at canonical Object.
	if !sourceInheritsObjectTextDefault(ctx.currentClass, ctx) {
		return nil
	}
	if target := resolveInvocationTarget(object, ctx, source); target != nil {
		selected, _ := findBestMethodForInvocationTarget(target, method, object.Parent().ChildByFieldName("arguments"), true, false, ctx, source)
		if selected != nil && selected.def != nil && len(selected.def.Parameters) == 0 {
			return nil
		}
	}
	return stdjavaCall(ctx, "ObjectDefaultFinalizeExecution", intrinsicExecutionExpr(ctx), ast.NewIdent(ShortName(ctx.className)))
}
