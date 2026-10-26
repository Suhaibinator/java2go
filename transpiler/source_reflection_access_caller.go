package transpiler

import (
	"fmt"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/token"
)

func reflectionLexicalCallerNode(invocation *sitter.Node) *sitter.Node {
	for node := invocation; node != nil; node = node.Parent() {
		switch node.Type() {
		case "class_declaration", "interface_declaration", "record_declaration", "enum_declaration", "annotation_type_declaration":
			return node
		case "object_creation_expression":
			if body := objectCreationClassBody(node); body != nil && threadSourceNodeWithin(invocation, body) {
				return node
			}
		}
	}
	return nil
}
func reflectionSourceCallerID(invocation *sitter.Node, ctx Ctx) (string, bool) {
	owner := ctx.currentClass
	if ctx.runtimeLocationProfile != sourceRuntimeLocationClasspath || owner == nil || owner.Class == nil || ctx.currentFile == nil || invocation == nil || ctx.currentFile.SourceInputSnapshot() == nil || ctx.currentFile.SourceInputSnapshot().Validate() != nil {
		return "", false
	}
	frame := ctx.lateSourceOwnership
	if frame == nil || frame != activeLateSourceOwnership || frame.file != ctx.currentFile || frame.invocation != ctx.syntaxInvocation || !frame.current() {
		return "", false
	}
	for current := owner; current != nil; current = current.Enclosing {
		if !frame.ownerCurrent(current) {
			return "", false
		}
	}
	declaration := owner.Class.DeclarationNode
	lexical := reflectionLexicalCallerNode(invocation)
	if declaration == nil || lexical == nil || !declaration.Equal(lexical) || !threadSourceNodeWithin(invocation, ctx.currentFile.ParsedRoot()) {
		return "", false
	}
	if declaring, owned := supplierSourceDeclarationCtx(owner, ctx); owned && declaring.currentFile == ctx.currentFile {
		return sourceClassRuntimeTypeID(owner, ctx), true
	}
	registration, present := frame.late[owner]
	if !present || !frame.registrationCurrent(owner, registration) {
		return "", false
	}
	if registration.local != nil && ctx.localClasses[registration.name] == registration.local {
		if registration.local.structName != owner.Class.Name || registration.local.dynamicTypeID != javaClassBinaryName(owner.Enclosing)+"$"+registration.local.structName {
			return "", false
		}
		// Actual local headers retain their source declaration. The ordinary header
		// checker needs no global registration when the synthesis witness owns it.
		if declaration.Type() != "class_declaration" {
			return "", false
		}
		if name := declaration.ChildByFieldName("name"); name == nil || name.Content(frame.file.Source) != owner.Class.OriginalName {
			return "", false
		}
		writtenSuperclass := ""
		if edge := declaration.ChildByFieldName("superclass"); edge != nil {
			types := supplierHeaderTypeNodes(edge)
			if len(types) != 1 {
				return "", false
			}
			writtenSuperclass = types[0].Content(frame.file.Source)
		}
		if writtenSuperclass != owner.Superclass {
			return "", false
		}
		var writtenInterfaces []string
		if edge := declaration.ChildByFieldName("interfaces"); edge != nil {
			for _, typ := range supplierHeaderTypeNodes(edge) {
				writtenInterfaces = append(writtenInterfaces, typ.Content(frame.file.Source))
			}
		}
		if len(writtenInterfaces) != len(owner.ImplementedInterfaces) {
			return "", false
		}
		for index := range writtenInterfaces {
			if writtenInterfaces[index] != owner.ImplementedInterfaces[index] {
				return "", false
			}
		}
		return registration.local.dynamicTypeID, true
	}
	if registration.anonymous == nil || registration.anonymous.structName != owner.Class.Name || ctx.anonymousClasses[registration.key] != registration.anonymous || declaration.Type() != "object_creation_expression" {
		return "", false
	}
	written := declaration.ChildByFieldName("type")
	if written == nil {
		return "", false
	}
	text := written.Content(frame.file.Source)
	matches := owner.Superclass == text && len(owner.ImplementedInterfaces) == 0 || owner.Superclass == "" && len(owner.ImplementedInterfaces) == 1 && owner.ImplementedInterfaces[0] == text
	if !matches {
		return "", false
	}
	return javaClassBinaryName(owner.Enclosing) + "$" + registration.anonymous.structName, true
}
func reflectionCallerExpression(invocation *sitter.Node, ctx Ctx) (ast.Expr, bool) {
	id, owned := reflectionSourceCallerID(invocation, ctx)
	if !owned {
		return nil, false
	}
	name := collisionSafeExecutionIdentifier("__java2goReflectionCallerFor"+fmt.Sprintf("%X", []byte(id)), ctx.currentClass)
	found := false
	if ctx.hoistedDecls != nil {
		for _, declaration := range *ctx.hoistedDecls {
			if declaration, ok := declaration.(*ast.GenDecl); ok {
				for _, spec := range declaration.Specs {
					if values, ok := spec.(*ast.ValueSpec); ok {
						for _, existing := range values.Names {
							found = found || existing.Name == name
						}
					}
				}
			}
		}
	}
	if !found {
		registration := &ast.FuncLit{Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.ExprStmt{X: stdjavaCall(ctx, "RegisterReflectionCallerNest", javaTypeIDLiteral(id, ctx), sourceReflectionNestHostTypeID(ctx.currentClass, ctx))},
			&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("true")}},
		}}}
		ctx.addHoistedDecl(&ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent(name)}, Values: []ast.Expr{&ast.CallExpr{Fun: registration}}}}})
	}
	return javaTypeIDLiteral(id, ctx), true
}

// Keep the supported caller proof separate from a nominal runtime type binding.
// A source homonym or donated ClassScope cannot select a privileged caller ID.
