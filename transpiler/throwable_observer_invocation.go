package transpiler

import sitter "github.com/smacker/go-tree-sitter"

// Select the inherited Throwable declaration only after Java owner/binder and
// source method resolution. A real source zero-formal observer keeps its body.
func builtinThrowableObserverSelected(object *sitter.Node, method string, ctx Ctx, source []byte) bool {
	if object == nil || invocationArgumentCount(object.Parent()) != 0 || (method != "getCause" && method != "getSuppressed" && method != "printStackTrace") {
		return false
	}
	declaring := ctx
	var receiverType string
	var known bool
	if object.Type() == "super" {
		if ctx.currentClass == nil {
			return false
		}
		declaring = classScopeCtx(ctx.currentClass, ctx)
		receiverType, known = ctx.currentClass.Superclass, true
	} else {
		receiverType, known = inferExprJavaType(object, ctx, source)
	}
	if !known || !isExceptionJavaType(declaring, receiverType) {
		return false
	}
	if target := resolveInvocationTarget(object, declaring, source); target != nil {
		selected, _ := findBestMethodForInvocationTarget(target, method, object.Parent().ChildByFieldName("arguments"), true, false, declaring, source)
		if !builtinMessageWinsSourceResolution(selected) {
			return false
		}
	}
	return true
}
