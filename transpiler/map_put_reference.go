package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// The Map contract does not expose the concrete Go selector of a specialized
// source override. Its erased invocation bridge performs the source bridge
// argument checks before entering that body, then projects the completed result.
func registerCanonicalMapPutIntrinsic() {
	registerIntrinsicOwner("java.util.Map", true)
	registerInstanceNodeIntrinsic("Map", "put", func(receiver ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if invocationArgumentCount(invocation) != 2 {
			return nil
		}
		args := intrinsicArgs(invocation.ChildByFieldName("object"), "put", source, ctx)
		call := stdjavaCall(ctx, "MapPutExecution", append([]ast.Expr{intrinsicExecutionExpr(ctx), receiver}, args...)...)
		logical, known := inferExprJavaType(invocation, ctx, source)
		if !known {
			logical = "java.lang.Object"
		}
		return mapErasedResultProjection(call, invocation, logical, ctx, source)
	})
}
