package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

func threadLocalElement(invocation *sitter.Node, ctx Ctx, source []byte) string {
	if elements := targetElementJavaTypes(invocation, ctx, source); len(elements) == 1 {
		return elements[0]
	}
	arg := invocationArgumentNode(invocation, 0)
	if arg != nil {
		if actual, ok := inferExprJavaType(arg, ctx, source); ok {
			_, elements := parseJavaTypeString(actual)
			if len(elements) == 1 {
				return elements[0]
			}
		}
		if arg.Type() == "lambda_expression" {
			body := arg.ChildByFieldName("body")
			if body != nil && body.Type() == "block" {
				body = callableReturnedExpression(body)
			}
			if actual, ok := inferExprJavaType(body, ctx, source); ok {
				return intrinsicReferenceJavaType(actual)
			}
		}
	}
	return "Object"
}
func init() {
	registerConstructorIntrinsic("ThreadLocal", func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		if len(typeArgs) == 0 {
			typeArgs = []ast.Expr{ast.NewIdent("any")}
		}
		return stdjavaGenericCall(ctx, "NewThreadLocal", typeArgs, nil)
	})
	registerStaticIntrinsic("ThreadLocal", "withInitial", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaGenericCall(ctx, "ThreadLocalWithInitial", ctx.intrinsicTypeArgs, args)
	})
	registerStaticIntrinsicTypeArgs("ThreadLocal", "withInitial", func(inv *sitter.Node, ctx Ctx, source []byte) []ast.Expr {
		return []ast.Expr{javaTypeStringToGoTypeExpr(threadLocalElement(inv, ctx, source), inScopeTypeParameters(ctx), ctx)}
	})
	registerStaticIntrinsicDerivedResultType("ThreadLocal", "withInitial", func(inv *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		return "ThreadLocal<" + threadLocalElement(inv, ctx, source) + ">", true
	})
	for java, goName := range map[string]string{"get": "Get", "set": "Set", "remove": "Remove"} {
		java, goName := java, goName
		registerInstanceIntrinsic("ThreadLocal", java, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			count := 0
			if java == "set" {
				count = 1
			}
			if len(args) != count {
				return nil
			}
			return methodCall(recv, goName, append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args...)...)
		})
	}
}

func isExternalSupplierType(javaType string, ctx Ctx) bool {
	base, _ := parseJavaTypeString(javaType)
	return stripJavaQualifier(base) == "Supplier" && resolveClassScopeByQualifiedName(ctx, base) == nil
}
