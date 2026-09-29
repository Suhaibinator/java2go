package transpiler

import (
	"go/ast"

	sitter "github.com/smacker/go-tree-sitter"
)

// Source overloads keep their normal dispatch. Canonical inherited zero-arity
// methods use the runtime's virtual protocol, with an explicit super call
// selecting only the inherited body rather than the most-derived override.
func builtinThrowableTextSelected(object *sitter.Node, method string, ctx Ctx, source []byte) bool {
	if (method != "toString" && method != "getLocalizedMessage") || object == nil || invocationArgumentCount(object.Parent()) != 0 || !builtinMessageReceiver(object, ctx, source) {
		return false
	}
	if target := resolveInvocationTarget(object, ctx, source); target != nil {
		if selected, _ := findBestMethodForInvocationTarget(target, method, object.Parent().ChildByFieldName("arguments"), true, false, ctx, source); !builtinMessageWinsSourceResolution(selected) {
			return false
		}
	}
	return true
}

func throwableTextInvocation(object *sitter.Node, method string, ctx Ctx, source []byte) ast.Expr {
	if !builtinThrowableTextSelected(object, method, ctx, source) {
		return nil
	}
	runtimeName := "ThrowableToString"
	if method == "getLocalizedMessage" {
		runtimeName = "ThrowableLocalizedMessage"
	}
	var receiver ast.Expr
	if object.Type() == "super" {
		runtimeName += "Default"
		receiver = ast.NewIdent(ShortName(ctx.className))
	} else {
		receiver = ParseExpr(object, source, ctx)
	}
	return stdjavaCall(ctx, runtimeName+"Execution", intrinsicExecutionExpr(ctx), receiver)
}
