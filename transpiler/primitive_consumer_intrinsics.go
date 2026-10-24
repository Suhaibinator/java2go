package transpiler

import "go/ast"

// Primitive consumers use the same SAM substitution, owner guards and argument
// conversion as the existing generic consumers. Register their Java primitive
// formal explicitly so lambdas and source/builtin method references agree.
func init() {
	for _, spec := range []struct {
		name      string
		signature builtinFunctionalInterface
	}{
		{"IntConsumer", builtinFunctionalInterface{parameterTypes: []string{"int"}, resultType: "void"}},
		{"ObjIntConsumer", builtinFunctionalInterface{typeParameters: []string{"T"}, parameterTypes: []string{"T", "int"}, resultType: "void"}},
	} {
		builtinFunctionalInterfaces[spec.name] = spec.signature
		intrinsicFunctionalMethodNames[spec.name] = "accept"
		registerInstanceIntrinsic(spec.name, "accept", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != len(spec.signature.parameterTypes) {
				return nil
			}
			return &ast.CallExpr{Fun: recv, Args: args}
		})
		registerInstanceIntrinsicResultType(spec.name, "accept", "void")
	}
}
