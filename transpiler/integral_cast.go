package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
)

// Java integral casts retain the low target-width bits. Express that operation
// before the Go conversion so both constant expressions and runtime values use
// the same rule: Go otherwise rejects overflowing constant conversions. The
// signed bias maps the masked value into the target's signed range, and the
// operand occurs once, preserving update/call side effects.
func lowerJavaIntegralCast(value ast.Expr, sourceType, targetType string) (ast.Expr, bool) {
	widths := map[string]uint{"byte": 8, "short": 16, "char": 16, "int": 32, "long": 64}
	if widths[sourceType] == 0 || widths[targetType] == 0 {
		return nil, false
	}
	target := ast.NewIdent(goPrimitiveConversionName(targetType))
	if targetType == "long" {
		return &ast.CallExpr{Fun: target, Args: []ast.Expr{value}}, true
	}
	width := widths[targetType]
	integer := func(value uint64) ast.Expr {
		return &ast.BasicLit{Kind: token.INT, Value: strconv.FormatUint(value, 10)}
	}
	widened := &ast.CallExpr{Fun: ast.NewIdent("int64"), Args: []ast.Expr{value}}
	var narrowed ast.Expr = &ast.BinaryExpr{X: widened, Op: token.AND, Y: integer((1 << width) - 1)}
	if targetType != "char" {
		bias := uint64(1) << (width - 1)
		narrowed = &ast.BinaryExpr{X: &ast.BinaryExpr{X: narrowed, Op: token.XOR, Y: integer(bias)}, Op: token.SUB, Y: integer(bias)}
	}
	return &ast.CallExpr{Fun: target, Args: []ast.Expr{narrowed}}, true
}
