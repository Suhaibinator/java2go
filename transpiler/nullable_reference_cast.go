package transpiler

import "go/ast"

// Java checkcast preserves all null references, including a typed nil behind
// Object. Go assertions reject nil interfaces, so normalize null before using
// the existing non-null assertion.
// A parameterized IIFE evaluates the operand exactly once without capturing
// caller identifiers. Nominal/subobject and array casts use their own helpers.
func nullableReferenceAssertion(value, target ast.Expr, targetJavaType string, ctx Ctx) ast.Expr {
	operand := ast.NewIdent("value")
	zero := zeroValueForType(target)
	return &ast.CallExpr{
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{
				Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{operand}, Type: ast.NewIdent("any")}}},
				Results: &ast.FieldList{List: []*ast.Field{{Type: target}}},
			},
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.IfStmt{Cond: stdjavaCall(ctx, "JavaReferenceEqual", operand, ast.NewIdent("nil")), Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{zero}}}}},
				&ast.ReturnStmt{Results: []ast.Expr{&ast.TypeAssertExpr{X: operand, Type: target}}},
			}},
		},
		Args: []ast.Expr{value},
	}
}
