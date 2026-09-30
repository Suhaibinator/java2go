package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/token"
)

func parseAssertionStatement(node *sitter.Node, source []byte, ctx Ctx) ast.Stmt {
	expressions := []*sitter.Node{}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		switch child.Type() {
		case "comment", "line_comment", "block_comment":
			continue
		}
		expressions = append(expressions, child)
	}
	if len(expressions) < 1 || len(expressions) > 2 {
		return nil
	}
	condition := parseJavaBooleanExpr(expressions[0], source, ctx)
	arguments := []ast.Expr{intrinsicExecutionExpr(ctx)}
	if len(expressions) == 2 {
		detail := ParseExpr(expressions[1], source, ctx)
		if isCharTypedExprNode(expressions[1], ctx, source) {
			detail = &ast.CallExpr{Fun: ast.NewIdent("string"), Args: []ast.Expr{detail}}
		}
		arguments = append(arguments, detail)
	}
	return &ast.IfStmt{Cond: &ast.BinaryExpr{X: stdjavaCall(ctx, "JavaAssertionsEnabled"), Op: token.LAND, Y: &ast.UnaryExpr{Op: token.NOT, X: condition}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: ast.NewIdent("panic"), Args: []ast.Expr{stdjavaCall(ctx, "NewAssertionFailure", arguments...)}}}}}}
}
