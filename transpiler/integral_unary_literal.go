package transpiler

import (
	"go/ast"
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// Fold only unary chains over integral literals, retaining their Java width.
// In particular, the positive decimal token for MIN_VALUE cannot be converted
// to a signed Go type before its minus sign. Nested negation also wraps in Java,
// whereas Go's arbitrary-precision constant arithmetic would overflow later.
func javaIntegralUnaryLiteral(node *sitter.Node, source []byte) ast.Expr {
	value, long, ok := integralUnaryLiteralValue(node, source)
	if !ok {
		return nil
	}
	constant := signedIntegerConstant(value)
	if !long {
		// Keep Java int literals assignable to a wider intrinsic parameter, as
		// ordinary integer literals already are. Folding has fixed their value
		// at the Java width; the consuming context supplies the Go storage type.
		return constant
	}
	return &ast.CallExpr{Fun: ast.NewIdent("int64"), Args: []ast.Expr{constant}}
}

func integralUnaryLiteralValue(node *sitter.Node, source []byte) (int64, bool, bool) {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return 0, false, false
	}
	if node.Type() == "unary_expression" && node.NamedChildCount() > 0 {
		value, long, ok := integralUnaryLiteralValue(node.NamedChild(int(node.NamedChildCount())-1), source)
		if !ok {
			return 0, false, false
		}
		switch node.Child(0).Content(source) {
		case "+":
		case "-":
			value = -value
		case "~":
			value = ^value
		default:
			return 0, false, false
		}
		if !long {
			value = int64(int32(value))
		}
		return value, long, true
	}
	base, prefix := 10, 0
	switch node.Type() {
	case "decimal_integer_literal":
	case "hex_integer_literal":
		base, prefix = 16, 2
	case "octal_integer_literal":
		base, prefix = 8, 1
	case "binary_integer_literal":
		base, prefix = 2, 2
	default:
		return 0, false, false
	}
	literal := strings.ReplaceAll(node.Content(source), "_", "")
	long := strings.HasSuffix(literal, "L") || strings.HasSuffix(literal, "l")
	if long {
		literal = literal[:len(literal)-1]
	}
	if prefix >= len(literal) {
		return 0, false, false
	}
	width := 32
	if long {
		width = 64
	}
	bits, err := strconv.ParseUint(literal[prefix:], base, width)
	if err != nil {
		return 0, false, false
	}
	value := int64(bits)
	if !long {
		value = int64(int32(bits))
	}
	return value, long, true
}
