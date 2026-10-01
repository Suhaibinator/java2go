package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// Called from registerByteBufferIntrinsics, so overload registration has no
// dependence on the lexical ordering of init functions.
func registerNIOHeapIntrinsics() {
	registerIntrinsicOwner("java.nio.ByteBuffer", true)
	for _, m := range []struct{ java, goName, result string }{
		{"capacity", "Capacity", "int"}, {"arrayOffset", "ArrayOffset", "int"},
		{"hasRemaining", "HasRemaining", "boolean"}, {"mark", "Mark", "ByteBuffer"},
		{"reset", "Reset", "ByteBuffer"}, {"rewind", "Rewind", "ByteBuffer"},
		{"duplicate", "Duplicate", "ByteBuffer"},
	} {
		registerInstanceIntrinsic("ByteBuffer", m.java, ioMethod(m.goName, 0))
		registerInstanceIntrinsicResultType("ByteBuffer", m.java, m.result)
	}
	registerInstanceIntrinsic("ByteBuffer", "limit", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) == 0 {
			return selectorCall(recv, "Limit", nil)
		}
		if len(args) == 1 {
			return selectorCall(recv, "SetLimit", args)
		}
		return nil
	})
	for _, name := range []string{"position", "limit"} {
		registerInstanceIntrinsicDerivedResultType("ByteBuffer", name, func(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
			if invocationArgumentCount(invocation) == 0 {
				return "int", true
			}
			if invocationArgumentCount(invocation) == 1 {
				return "java.nio.ByteBuffer", true
			}
			return "", false
		})
	}
	registerInstanceIntrinsic("ByteBuffer", "slice", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 && len(args) != 2 {
			return nil
		}
		return selectorCall(recv, "Slice", args)
	})
	registerInstanceIntrinsicResultType("ByteBuffer", "slice", "ByteBuffer")
	registerInstanceNodeIntrinsic("ByteBuffer", "get", lowerNIOHeapGet)
	registerInstanceIntrinsicDerivedResultType("ByteBuffer", "get", func(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		count := invocationArgumentCount(invocation)
		if count == 0 {
			return "byte", true
		}
		if count == 1 {
			kind := nioHeapArgumentKind(invocation, 0, ctx, source)
			if kind == "index" || kind == "byte" {
				return "byte", true
			}
		}
		if count == 1 || count == 3 {
			return "java.nio.ByteBuffer", true
		}
		return "", false
	})
	registerInstanceNodeIntrinsic("ByteBuffer", "put", lowerNIOHeapPut)
	registerInstanceIntrinsicResultType("ByteBuffer", "put", "ByteBuffer")
}

func nioHeapArgumentKind(invocation *sitter.Node, index int, ctx Ctx, source []byte) string {
	actual, known := inferExprJavaType(invocationArgumentNode(invocation, index), ctx, source)
	if !known {
		return ""
	}
	base, rank := javaArrayTypeParts(actual)
	if base == "byte" && rank == 1 {
		return "array"
	}
	if owner, admitted := canonicalIntrinsicOwner(actual, ctx); admitted && owner == "java.nio.ByteBuffer" {
		return "buffer"
	}
	if actual == "byte" {
		return "byte"
	}
	if actual == "int" || actual == "short" || actual == "char" {
		return "index"
	}
	return ""
}

func lowerNIOHeapGet(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	count := invocationArgumentCount(invocation)
	name := "Get"
	expected := []string{}
	switch count {
	case 0:
	case 1:
		switch nioHeapArgumentKind(invocation, 0, ctx, source) {
		case "index", "byte":
			expected = []string{"int"}
		case "array":
			name, expected = "GetInto", []string{"byte[]"}
		default:
			return nil
		}
	case 3:
		if nioHeapArgumentKind(invocation, 0, ctx, source) != "array" {
			return nil
		}
		name, expected = "GetInto", []string{"byte[]", "int", "int"}
	default:
		return nil
	}
	args := parseArgumentListWithExpectedTypes(invocation.ChildByFieldName("arguments"), source, ctx, expected)
	return selectorCall(recv, name, args)
}

func lowerNIOHeapPut(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	count := invocationArgumentCount(invocation)
	name := ""
	var expected []string
	switch count {
	case 1:
		switch nioHeapArgumentKind(invocation, 0, ctx, source) {
		case "byte":
			name, expected = "Put", []string{"byte"}
		case "array":
			name, expected = "PutArray", []string{"byte[]"}
		case "buffer":
			name, expected = "PutBuffer", []string{"java.nio.ByteBuffer"}
		default:
			return nil
		}
	case 2:
		name, expected = "PutAt", []string{"int", "byte"}
	case 3:
		if nioHeapArgumentKind(invocation, 0, ctx, source) != "array" {
			return nil
		}
		name, expected = "PutArray", []string{"byte[]", "int", "int"}
	default:
		return nil
	}
	args := parseArgumentListWithExpectedTypes(invocation.ChildByFieldName("arguments"), source, ctx, expected)
	return selectorCall(recv, name, args)
}
