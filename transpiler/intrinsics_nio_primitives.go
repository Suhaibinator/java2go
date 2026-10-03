package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// Register explicitly after heap overloads; no init-order dependency.
func registerNIOPrimitiveIntrinsics() {
	registerIntrinsicOwner("java.nio.ByteOrder", true)
	for _, name := range []string{"BIG_ENDIAN", "LITTLE_ENDIAN"} {
		field := name
		registerStaticFieldIntrinsic("ByteOrder", field, func(ctx Ctx) ast.Expr { return stdjavaQualifiedExpr("ByteOrder"+field, ctx) })
		registerStaticFieldIntrinsicResultType("ByteOrder", field, "java.nio.ByteOrder")
	}
	registerInstanceIntrinsic("ByteOrder", "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return methodCall(recv, "StringJava2goExecution", intrinsicExecutionExpr(ctx))
	})
	registerInstanceIntrinsicResultType("ByteOrder", "toString", "java.lang.String")
	registerInstanceIntrinsic("ByteBuffer", "order", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) == 0 {
			return selectorCall(recv, "Order", nil)
		}
		if len(args) == 1 {
			return selectorCall(recv, "SetOrder", args)
		}
		return nil
	})
	registerInstanceIntrinsicDerivedResultType("ByteBuffer", "order", func(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		if invocationArgumentCount(invocation) == 0 {
			return "java.nio.ByteOrder", true
		}
		if invocationArgumentCount(invocation) == 1 {
			return "java.nio.ByteBuffer", true
		}
		return "", false
	})
	for _, spec := range []struct{ suffix, primitive string }{
		{"Short", "short"}, {"Int", "int"}, {"Long", "long"}, {"Float", "float"}, {"Double", "double"},
	} {
		suffix, primitive := spec.suffix, spec.primitive
		registerInstanceNodeIntrinsic("ByteBuffer", "get"+suffix, func(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			count := invocationArgumentCount(invocation)
			if count != 0 && count != 1 {
				return nil
			}
			expected := []string{}
			if count == 1 {
				expected = []string{"int"}
			}
			args := parseArgumentListWithExpectedTypes(invocation.ChildByFieldName("arguments"), source, ctx, expected)
			return selectorCall(recv, "Get"+suffix, args)
		})
		registerInstanceIntrinsicResultType("ByteBuffer", "get"+suffix, primitive)
		registerInstanceNodeIntrinsic("ByteBuffer", "put"+suffix, func(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			count := invocationArgumentCount(invocation)
			name, expected := "Put"+suffix, []string{primitive}
			if count == 2 {
				name, expected = name+"At", []string{"int", primitive}
			} else if count != 1 {
				return nil
			}
			args := parseArgumentListWithExpectedTypes(invocation.ChildByFieldName("arguments"), source, ctx, expected)
			return selectorCall(recv, name, args)
		})
		registerInstanceIntrinsicResultType("ByteBuffer", "put"+suffix, "java.nio.ByteBuffer")
	}
}
