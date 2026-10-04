package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

// Emitter and result metadata share the same owner, declaration and signature
// proof. Source declarations and binders never inherit a simple-name service.
func nioHeapCanonicalInvocation(node *sitter.Node, owner string, ctx Ctx, source []byte) bool {
	if node == nil {
		return false
	}
	object := node.ChildByFieldName("object")
	actual, known := inferExprJavaType(object, ctx, source)
	if !known {
		return false
	}
	element, rank := javaArrayTypeParts(actual)
	base, args := parseJavaTypeString(strings.TrimSpace(element))
	if rank != 0 || len(args) != 0 || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return false
	}
	canonical, ok := canonicalIntrinsicOwner(base, ctx)
	return ok && canonical == owner
}
func nioHeapSignatureApplicable(node *sitter.Node, owner string, expected []string, ctx Ctx, source []byte) bool {
	if !nioHeapCanonicalInvocation(node, owner, ctx, source) || invocationArgumentCount(node) != len(expected) {
		return false
	}
	for i, typ := range expected {
		if typ == "java.nio.ByteBuffer" {
			argument := invocationArgumentNode(node, i)
			actual, known := inferExprJavaType(argument, ctx, source)
			element, rank := javaArrayTypeParts(actual)
			base, args := parseJavaTypeString(element)
			owner, canonical := canonicalIntrinsicOwner(base, ctx)
			if !known || rank != 0 || len(args) != 0 || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil || !canonical || owner != "java.nio.ByteBuffer" {
				return false
			}
		}
		if !intrinsicInvocationConversionApplicable(invocationArgumentNode(node, i), typ, ctx, source) {
			return false
		}
	}
	return true
}
func nioHeapPutSignature(node *sitter.Node, ctx Ctx, source []byte) (string, []string, bool) {
	if invocationArgumentCount(node) == 1 || invocationArgumentCount(node) == 4 {
		index := 0
		if invocationArgumentCount(node) == 4 {
			index = 1
		}
		arg := unwrapParenthesizedExpressionNode(invocationArgumentNode(node, index))
		if arg != nil && arg.Type() == "null_literal" {
			return "", nil, false
		}
	}
	for _, spec := range []struct {
		name string
		args []string
	}{
		{"Put", []string{"byte"}}, {"PutAt", []string{"int", "byte"}},
		{"PutArray", []string{"byte[]"}}, {"PutArray", []string{"byte[]", "int", "int"}},
		{"PutBuffer", []string{"java.nio.ByteBuffer"}},
		{"PutArrayAtWhole", []string{"int", "byte[]"}},
		{"PutArrayAt", []string{"int", "byte[]", "int", "int"}},
		{"PutBufferAt", []string{"int", "java.nio.ByteBuffer", "int", "int"}},
	} {
		if nioHeapSignatureApplicable(node, "java.nio.ByteBuffer", spec.args, ctx, source) {
			return spec.name, spec.args, true
		}
	}
	return "", nil, false
}
func registerNIOReadonlyIntrinsics() {
	for _, spec := range []struct{ java, goName, result string }{{"asReadOnlyBuffer", "AsReadOnlyBuffer", "java.nio.ByteBuffer"}, {"isReadOnly", "IsReadOnly", "boolean"}} {
		name, goName, result := spec.java, spec.goName, spec.result
		registerInstanceNodeIntrinsic("ByteBuffer", name, func(recv ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !nioHeapSignatureApplicable(node, "java.nio.ByteBuffer", nil, ctx, source) {
				return unsupportedIntrinsicValue(node, result, source, ctx)
			}
			return selectorCall(recv, goName, nil)
		})
		registerInstanceIntrinsicDerivedResultType("ByteBuffer", name, func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
			if nioHeapSignatureApplicable(node, "java.nio.ByteBuffer", nil, ctx, source) {
				return result, true
			}
			return "", false
		})
	}
	registerInstanceIntrinsicDerivedResultType("ByteBuffer", "put", func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		if _, _, ok := nioHeapPutSignature(node, ctx, source); ok {
			return "java.nio.ByteBuffer", true
		}
		return "", false
	})
}
func registerReadonlyFileChannelReadIntrinsic() {
	registerIntrinsicOwner("java.nio.channels.FileChannel", true)
	applicable := func(node *sitter.Node, ctx Ctx, source []byte) bool {
		arg := unwrapParenthesizedExpressionNode(invocationArgumentNode(node, 0))
		return arg != nil && arg.Type() != "null_literal" && nioHeapSignatureApplicable(node, "java.nio.channels.FileChannel", []string{"java.nio.ByteBuffer"}, ctx, source)
	}
	registerInstanceNodeIntrinsic("FileChannel", "read", func(recv ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if !applicable(node, ctx, source) {
			return unsupportedIntrinsicValue(node, "int", source, ctx)
		}
		args := parseArgumentListWithExpectedTypes(node.ChildByFieldName("arguments"), source, ctx, []string{"java.nio.ByteBuffer"})
		return stdjavaCall(ctx, "FileChannelReadExecution", intrinsicExecutionExpr(ctx), recv, args[0])
	})
	registerInstanceIntrinsicDerivedResultType("FileChannel", "read", func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		if applicable(node, ctx, source) {
			return "int", true
		}
		return "", false
	})
}

func registerNIOCharIntrinsics() {
	for _, name := range []string{"getChar", "putChar"} {
		result := "char"
		if name == "putChar" {
			result = "java.nio.ByteBuffer"
		}
		selectSignature := func(node *sitter.Node, ctx Ctx, source []byte) (string, []string, bool) {
			if name == "getChar" {
				for _, args := range [][]string{nil, {"int"}} {
					if nioHeapSignatureApplicable(node, "java.nio.ByteBuffer", args, ctx, source) {
						return "GetChar", args, true
					}
				}
			} else {
				if nioHeapSignatureApplicable(node, "java.nio.ByteBuffer", []string{"char"}, ctx, source) {
					return "PutChar", []string{"char"}, true
				}
				if nioHeapSignatureApplicable(node, "java.nio.ByteBuffer", []string{"int", "char"}, ctx, source) {
					return "PutCharAt", []string{"int", "char"}, true
				}
			}
			return "", nil, false
		}
		registerInstanceNodeIntrinsic("ByteBuffer", name, func(recv ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			goName, args, ok := selectSignature(node, ctx, source)
			if !ok {
				return unsupportedIntrinsicValue(node, result, source, ctx)
			}
			return selectorCall(recv, goName, parseArgumentListWithExpectedTypes(node.ChildByFieldName("arguments"), source, ctx, args))
		})
		registerInstanceIntrinsicDerivedResultType("ByteBuffer", name, func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
			if _, _, ok := selectSignature(node, ctx, source); ok {
				return result, true
			}
			return "", false
		})
	}
}
