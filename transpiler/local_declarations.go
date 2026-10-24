package transpiler

import (
	"go/ast"
	"go/token"

	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

// Each Java declarator becomes a separate Go var specification. The enclosing
// declaration group keeps their original lexical scope, and later initializers
// can read earlier variables. A parallel Go assignment would change that scope
// and evaluation behavior; a block would hide all variables from later code.
func parseLocalVariableDeclaration(node *sitter.Node, source []byte, ctx Ctx) ast.Stmt {
	declarators := nodeutil.VariableDeclarators(node)
	if len(declarators) == 1 {
		return parseLocalVariableDeclarator(node, declarators[0], source, ctx)
	}
	group := &ast.GenDecl{Tok: token.VAR, Lparen: token.Pos(1)}
	for _, declarator := range declarators {
		statement := parseLocalVariableDeclarator(node, declarator, source, ctx)
		switch declaration := statement.(type) {
		case *ast.DeclStmt:
			group.Specs = append(group.Specs, declaration.Decl.(*ast.GenDecl).Specs...)
		case *ast.AssignStmt:
			names := make([]*ast.Ident, len(declaration.Lhs))
			for i, name := range declaration.Lhs {
				names[i] = name.(*ast.Ident)
			}
			group.Specs = append(group.Specs, &ast.ValueSpec{Names: names, Values: declaration.Rhs})
		}
	}
	return &ast.DeclStmt{Decl: group}
}
