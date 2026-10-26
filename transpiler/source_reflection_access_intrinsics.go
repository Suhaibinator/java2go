package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

func registerReflectionAccessIntrinsics() {
	for _, receiver := range []string{"AccessibleObject", "Field", "Method", "Constructor"} {
		for _, name := range []string{"canAccess", "setAccessible", "trySetAccessible", "isAccessible"} {
			name := name
			registerInstanceNodeIntrinsic(receiver, name, func(value ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
				arguments := intrinsicArgs(node.ChildByFieldName("object"), name, source, ctx)
				arity := 1
				if name == "trySetAccessible" || name == "isAccessible" {
					arity = 0
				}
				if len(arguments) != arity {
					return unsupportedIntrinsicValue(node, "boolean", source, ctx)
				}
				if name == "isAccessible" {
					return stdjavaCall(ctx, "AccessibleObjectIsAccessible", value)
				}
				caller, known := reflectionCallerExpression(node, ctx)
				if !known {
					if ctx.runtimeLocationProfile == sourceRuntimeLocationClasspath {
						return unsupportedIntrinsicValue(node, "java.lang.Object", source, ctx)
					}
					if receiver != "AccessibleObject" && name != "trySetAccessible" {
						return selectorCall(value, strings.ToUpper(name[:1])+name[1:], arguments)
					}
					caller = javaTypeIDLiteral("", ctx)
				}
				operation := map[string]string{"canAccess": "AccessibleObjectCanAccessExecution", "setAccessible": "AccessibleObjectSetAccessibleExecution", "trySetAccessible": "AccessibleObjectTrySetAccessibleExecution"}[name]
				return stdjavaCall(ctx, operation, append([]ast.Expr{intrinsicExecutionExpr(ctx), caller, value}, arguments...)...)
			})
			result := "boolean"
			if name == "setAccessible" {
				result = "void"
			}
			registerInstanceIntrinsicResultType(receiver, name, result)
		}
	}
	registerStaticNodeIntrinsic("AccessibleObject", "setAccessible", func(node *sitter.Node, _ []ast.Expr, ctx Ctx, source []byte) ast.Expr {
		arguments := intrinsicArgs(node.ChildByFieldName("object"), "setAccessible", source, ctx)
		if len(arguments) != 2 {
			return unsupportedIntrinsicValue(node, "java.lang.Object", source, ctx)
		}
		caller, owned := reflectionCallerExpression(node, ctx)
		if !owned {
			return unsupportedIntrinsicValue(node, "java.lang.Object", source, ctx)
		}
		return stdjavaCall(ctx, "AccessibleObjectSetAccessibleArrayExecution", append([]ast.Expr{intrinsicExecutionExpr(ctx), caller}, arguments...)...)
	})
	registerStaticIntrinsicResultType("AccessibleObject", "setAccessible", "void")
	for _, name := range []string{"getModifiers", "getDeclaringClass", "getName", "isSynthetic"} {
		name := name
		registerInstanceIntrinsic("Member", name, func(value ast.Expr, arguments []ast.Expr, ctx Ctx) ast.Expr {
			if len(arguments) != 0 {
				return nil
			}
			if name == "getName" {
				return stdjavaCall(ctx, "ReflectionMemberName", value)
			}
			return selectorCall(value, strings.ToUpper(name[:1])+name[1:], nil)
		})
		result := map[string]string{"getModifiers": "int", "getDeclaringClass": "java.lang.Class<?>", "getName": "java.lang.String", "isSynthetic": "boolean"}[name]
		registerInstanceIntrinsicResultType("Member", name, result)
	}
}
