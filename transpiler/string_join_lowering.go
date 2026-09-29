package transpiler

import (
	"go/ast"

	sitter "github.com/smacker/go-tree-sitter"
)

// String.join overloads differ in validation and iteration order. Select using
// the declared Java type, never the dynamic Go value (especially for null).
// The caller already parsed arguments once, in their original Java order.
func lowerStringJoin(invocation *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	if len(args) == 0 {
		return unsupportedIntrinsicValue(invocation, "String", source, ctx)
	}
	helper := "StringJoinValuesExecution"
	if len(args) == 2 {
		argument := invocationArgumentNode(invocation, 1)
		javaType, known := inferExprJavaType(argument, ctx, source)
		if !known {
			return unsupportedIntrinsicValue(invocation, "String", source, ctx)
		}
		base, rank := javaArrayTypeParts(javaType)
		switch {
		case rank == 1:
			if _, primitive := javaPrimitiveArrayComponent(javaType); primitive {
				return unsupportedIntrinsicValue(invocation, "String", source, ctx)
			}
			helper = "StringJoinArrayExecution"
		case rank != 0:
			return unsupportedIntrinsicValue(invocation, "String", source, ctx)
		default:
			base, _ = parseJavaTypeString(base)
			if erased, bounded := javaTypeParameterErasure(base, ctx); bounded {
				base, _ = parseJavaTypeString(erased)
			}
			name := stripJavaQualifier(base)
			// Source collection declarations require the source iterator bridge; they
			// must not inherit the runtime mapping simply by sharing a short name.
			builtin := resolveClassScopeByQualifiedName(ctx, base) == nil
			if builtin && (containsString(listTypeNames, name) || containsString(setTypeNames, name) || name == "Collection" || name == "Iterable") {
				helper = "StringJoinIterableExecution"
			} else if !intrinsicInvocationConversionApplicable(argument, "java.lang.CharSequence", ctx, source) {
				return unsupportedIntrinsicValue(invocation, "String", source, ctx)
			}
		}
	}
	return stdjavaCall(ctx, helper, append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args...)...)
}
