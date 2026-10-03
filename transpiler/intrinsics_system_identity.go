package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

func init() {
	registerIntrinsicOwner("java.lang.System", true)
	registerStaticIntrinsic("System", "identityHashCode", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "SystemIdentityHashCode", args[0])
	})
	registerStaticIntrinsicExpectedArguments("System", "identityHashCode", func(invocation *sitter.Node, _ Ctx, _ []byte) []string {
		if invocationArgumentCount(invocation) != 1 {
			return nil
		}
		return []string{"java.lang.Object"}
	})
	registerStaticIntrinsicResultType("System", "identityHashCode", "int")
	registerStaticIntrinsicImportSignature("System", "identityHashCode", "java.lang.Object")
}
