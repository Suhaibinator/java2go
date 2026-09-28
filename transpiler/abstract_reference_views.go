package transpiler

import (
	"go/ast"
	"go/token"

	"github.com/NickyBoy89/java2go/symbol"
)

// Non-generic abstract references use companion interfaces. Generic abstract
// references retain a physical *Base[T] view, whose dispatch field points at the
// most-derived receiver; invoking the base's method directly would hit its stub.
func abstractClassUsesInterfaceView(scope *symbol.ClassScope) bool {
	return scope != nil && scope.IsAbstract && len(scope.TypeParameters) == 0
}

// Selecting a superclass subobject is a reference conversion, so it must retain
// null and evaluate its operand only once.
func nullableEmbeddedSuperclassView(value ast.Expr, actualType, expectedType, field string, ctx Ctx) ast.Expr {
	actualGoType := javaTypeStringToGoTypeExpr(actualType, inScopeTypeParameters(ctx), ctx)
	expectedGoType := javaTypeStringToGoTypeExpr(expectedType, inScopeTypeParameters(ctx), ctx)
	parameter := ast.NewIdent("__java2goBaseValue")
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{parameter}, Type: actualGoType}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: expectedGoType}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.IfStmt{
				Cond: &ast.BinaryExpr{X: parameter, Op: token.EQL, Y: ast.NewIdent("nil")},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("nil")}}}},
			},
			&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: parameter, Sel: ast.NewIdent(field)}}},
		}},
	}, Args: []ast.Expr{value}}
}
