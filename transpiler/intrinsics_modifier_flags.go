package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

func init() {
	registerIntrinsicOwner("java.lang.reflect.Modifier", true)
	for _, query := range []struct{ java, helper string }{
		{"isPublic", "ModifierIsPublic"},
		{"isPrivate", "ModifierIsPrivate"},
		{"isProtected", "ModifierIsProtected"},
		{"isStatic", "ModifierIsStatic"},
		{"isFinal", "ModifierIsFinal"},
		{"isSynchronized", "ModifierIsSynchronized"},
		{"isVolatile", "ModifierIsVolatile"},
		{"isTransient", "ModifierIsTransient"},
		{"isNative", "ModifierIsNative"},
		{"isInterface", "ModifierIsInterface"},
		{"isAbstract", "ModifierIsAbstract"},
		{"isStrict", "ModifierIsStrict"},
	} {
		registerStaticNodeIntrinsic("Modifier", query.java, func(node *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
			if len(args) != 1 || !intrinsicInvocationConversionApplicable(invocationArgumentNode(node, 0), "int", ctx, source) {
				return unsupportedIntrinsicValue(node, "boolean", source, ctx)
			}
			return stdjavaCall(ctx, query.helper, args[0])
		})
		// Method-reference lowering consumes the same signature without a Java
		// invocation node; its arguments have already been adapted to int.
		registerStaticIntrinsic("Modifier", query.java, func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 1) {
				return nil
			}
			return stdjavaCall(ctx, query.helper, args[0])
		})
		registerStaticIntrinsicExpectedArguments("Modifier", query.java, func(node *sitter.Node, ctx Ctx, source []byte) []string {
			if invocationArgumentCount(node) != 1 || !intrinsicInvocationConversionApplicable(invocationArgumentNode(node, 0), "int", ctx, source) {
				return nil
			}
			return []string{"int"}
		})
		registerStaticIntrinsicResultType("Modifier", query.java, "boolean")
		registerStaticIntrinsicImportSignature("Modifier", query.java, "int")
	}
	for _, flag := range []struct{ java, helper string }{
		{"PUBLIC", "ModifierPublic"},
		{"PRIVATE", "ModifierPrivate"},
		{"PROTECTED", "ModifierProtected"},
		{"STATIC", "ModifierStatic"},
		{"FINAL", "ModifierFinal"},
		{"SYNCHRONIZED", "ModifierSynchronized"},
		{"VOLATILE", "ModifierVolatile"},
		{"TRANSIENT", "ModifierTransient"},
		{"NATIVE", "ModifierNative"},
		{"INTERFACE", "ModifierInterface"},
		{"ABSTRACT", "ModifierAbstract"},
		{"STRICT", "ModifierStrict"},
	} {
		registerStaticFieldIntrinsic("Modifier", flag.java, func(ctx Ctx) ast.Expr { return stdjavaQualifiedExpr(flag.helper, ctx) })
		registerStaticFieldIntrinsicResultType("Modifier", flag.java, "int")
	}
}
