package transpiler

import "go/ast"

func init() {
	registerConstructorIntrinsic("BitSet", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) > 1 {
			return nil
		}
		return stdjavaCall(ctx, "NewBitSet", args...)
	})
	for _, method := range []struct {
		java, goName, result string
		args                 int
	}{
		{"set", "Set", "void", 1}, {"get", "Get", "boolean", 1}, {"clone", "Clone", "Object", 0},
		{"length", "Length", "int", 0}, {"size", "Size", "int", 0}, {"cardinality", "Cardinality", "int", 0},
	} {
		registerInstanceIntrinsic("BitSet", method.java, ioMethod(method.goName, method.args))
		registerInstanceIntrinsicResultType("BitSet", method.java, method.result)
	}
}
