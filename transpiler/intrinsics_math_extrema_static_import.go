package transpiler

import (
	"go/ast"

	sitter "github.com/smacker/go-tree-sitter"
)

func init() {
	// Java Math declares the same four primitive overloads for min and max.
	// Keep import applicability in the existing overload resolver; lowering
	// and numeric result/argument promotion remain shared with qualified calls.
	for _, method := range []string{"max", "min"} {
		registerStaticIntrinsicTypeArgs("Math", method, mathExtremaPrimitiveTypeArgs)
		for _, primitive := range []string{"int", "long", "float", "double"} {
			registerStaticIntrinsicImportSignature("Math", method, primitive, primitive)
		}
	}
}

// Explicit Go helper instantiation preserves the selected Java primitive width
// when all arguments remain untyped constants in emitted Go. Nonprimitive paths
// retain their previous lowering; this helper does not admit new invocations.
func mathExtremaPrimitiveTypeArgs(invocation *sitter.Node, ctx Ctx, source []byte) []ast.Expr {
	if invocation == nil || invocationArgumentCount(invocation) != 2 {
		return nil
	}
	for index := 0; index < 2; index++ {
		actual, known := inferExprJavaType(invocationArgumentNode(invocation, index), ctx, source)
		if !known {
			return nil
		}
		switch actual {
		case "byte", "short", "char", "int", "long", "float", "double":
		default:
			return nil
		}
	}
	primitive := intrinsicMathParameterJavaType(invocation, ctx, source)
	return []ast.Expr{javaTypeStringToGoTypeExpr(primitive, inScopeTypeParameters(ctx), ctx)}
}
