package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

func init() {
	for _, owner := range listTypeNames {
		registerInstanceNodeIntrinsic(owner, "subList", func(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if invocationArgumentCount(invocation) != 2 {
				return nil
			}
			args := intrinsicArgs(invocation.ChildByFieldName("object"), "subList", source, ctx)
			if rawListReceiver(invocation, ctx, source) {
				return stdjavaCall(ctx, "CollectionListSubListExecution", append([]ast.Expr{intrinsicExecutionExpr(ctx), recv}, args...)...)
			}
			return methodCall(recv, "SubList", args...)
		})
		registerInstanceIntrinsicDerivedResultType(owner, "subList", func(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
			elements := receiverElementJavaTypes(invocation.ChildByFieldName("object"), ctx, source)
			if len(elements) == 1 {
				return "java.util.List<" + elements[0] + ">", true
			}
			return "java.util.List", true
		})
	}
}
