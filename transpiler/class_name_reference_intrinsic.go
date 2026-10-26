package transpiler

import "go/ast"

// The ordinary intrinsic table serves both calls and bound/unbound method
// references. Its receiver resolution excludes source classes named Class.
func classJavaNameIntrinsic(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
	if len(args) != 0 {
		return nil
	}
	return stdjavaCall(ctx, "ClassJavaName", recv)
}

func classJavaSimpleNameIntrinsic(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
	if len(args) != 0 {
		return nil
	}
	return stdjavaCall(ctx, "ClassJavaSimpleName", recv)
}
