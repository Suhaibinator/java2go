package transpiler

import (
	"go/ast"
	"go/token"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Only implicit and compact canonical constructors perform the implicit final
// component assignments. They occur after normal body completion, so a thrown
// exception leaves them unperformed. Explicit canonical bodies own assignments.
func recordCanonicalTail(ctx Ctx, receiver string) []ast.Stmt {
	components, known := recordConstructorComponents(ctx.currentClass)
	if !known || ctx.localScope == nil || len(ctx.localScope.Parameters) != len(components) {
		var node *sitter.Node
		var source []byte
		if ctx.currentClass != nil && ctx.currentClass.Class != nil {
			node = ctx.currentClass.Class.DeclarationNode
		}
		if ctx.currentFile != nil {
			source = ctx.currentFile.Source
		}
		diagnostic := reportUnsupported("record canonical constructor component ownership", node, source, ctx)
		return []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: ast.NewIdent("panic"), Args: []ast.Expr{metadataString(diagnostic.String())}}}}
	}
	var statements []ast.Stmt
	for index, component := range components {
		statements = append(statements, &ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent(receiver), Sel: ast.NewIdent(component.field.Name)}}, Tok: token.ASSIGN, Rhs: []ast.Expr{ast.NewIdent(ctx.localScope.Parameters[index].Name)}})
	}
	return statements
}

func buildRecordImplicitConstructorDecls(node *sitter.Node, source []byte, ctx Ctx, method *symbol.Definition) []ast.Decl {
	return buildSourceConstructorDecls(node, source, ctx, constructorLoweringOptions{
		definition: method, parameterNode: node.ChildByFieldName("parameters"), bodyNodeSupplied: true,
		canonicalTail: recordCanonicalTail,
	})
}

func buildRecordCompactConstructorDecls(node *sitter.Node, source []byte, ctx Ctx) []ast.Decl {
	method := recordCanonicalConstructor(ctx.currentClass, ctx)
	if method == nil || method.DeclarationNode == nil || !method.DeclarationNode.Equal(node) {
		return []ast.Decl{unsupportedDeclStub(reportUnsupported("record compact canonical constructor ownership", node, source, ctx))}
	}
	return buildSourceConstructorDecls(node, source, ctx, constructorLoweringOptions{
		definition: method, parameterNode: ctx.currentClass.Class.DeclarationNode.ChildByFieldName("parameters"),
		canonicalTail: recordCanonicalTail,
	})
}

// Source allocation and method-reference dispatch may use hidden execution
// entries only for implementations established by this owned record plan.
// The compatibility Go wrappers still start a fresh execution for Go callers.
func recordConstructorExecutionReady(method *symbol.Definition, scope *symbol.ClassScope) bool {
	if !sourceReflectionRecord(scope) {
		return false
	}
	canonical := recordCanonicalConstructor(scope, Ctx{})
	if canonical == nil {
		return false
	}
	if method == nil {
		return canonical.DeclarationNode == nil && len(canonical.Parameters) == 0
	}
	if !method.Constructor {
		return false
	}
	if method.DeclarationNode == nil {
		return method == canonical
	}
	return recordWrittenConstructorOwned(scope, method)
}
