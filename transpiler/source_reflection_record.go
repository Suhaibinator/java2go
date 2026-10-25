package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func sourceReflectionRecord(scope *symbol.ClassScope) bool {
	return scope != nil && scope.Class != nil && scope.Class.DeclarationNode != nil && scope.Class.DeclarationNode.Type() == "record_declaration"
}

// Record backing fields are implicit declarations derived from actual header
// components. Names/types must agree with the retained parser symbols.
func sourceReflectionRecordComponent(scope *symbol.ClassScope, field *symbol.Definition) *sitter.Node {
	if !sourceReflectionRecord(scope) || field == nil || field.DeclarationNode != nil || field.IsStatic {
		return nil
	}
	file := findFileScopeForClassScope(scope)
	if file == nil || file.SourceInputSnapshot() == nil || file.SourceInputSnapshot().Validate() != nil || !threadSourceNodeWithin(scope.Class.DeclarationNode, file.ParsedRoot()) {
		return nil
	}
	name := scope.Class.DeclarationNode.ChildByFieldName("name")
	if name == nil || name.Content(file.Source) != scope.Class.OriginalName {
		return nil
	}
	parameters := scope.Class.DeclarationNode.ChildByFieldName("parameters")
	for _, component := range nodeutil.NamedChildrenOf(parameters) {
		name, typ := component.ChildByFieldName("name"), component.ChildByFieldName("type")
		if component.Type() == "formal_parameter" && name != nil && typ != nil && name.Content(file.Source) == field.OriginalName && typ.Content(file.Source) == field.OriginalType {
			return component
		}
	}
	return nil
}

func sourceReflectionRecordGetter(scope *symbol.ClassScope, field *symbol.Definition, ctx Ctx) ast.Decl {
	receiver := ast.NewIdent("receiver")
	physical := javaTypeStringToGoTypeExpr(field.OriginalType, scope.TypeParameterNames(), classScopeCtx(scope, ctx))
	return &ast.FuncDecl{Name: ast.NewIdent(sourceReflectionFieldAccessorName(scope, field, false)), Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{receiver}, Type: classSubobjectDeclarationPointerType(scope, ctx)}}}, Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}, Results: &ast.FieldList{List: []*ast.Field{{Type: physical}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(field.Name)}}}}}}
}
func sourceReflectionUnavailableCallback(ctx Ctx, parameters *ast.FieldList, results *ast.FieldList, message string) ast.Expr {
	return &ast.FuncLit{Type: &ast.FuncType{Params: parameters, Results: results}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: ast.NewIdent("panic"), Args: []ast.Expr{stdjavaCall(ctx, "NewUnsupportedOperationException", metadataString(message))}}}}}}
}
func sourceReflectionRecordSetUnavailable(ctx Ctx) ast.Expr {
	return sourceReflectionUnavailableCallback(ctx, &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx), {Names: []*ast.Ident{ast.NewIdent("receiver"), ast.NewIdent("value")}, Type: ast.NewIdent("any")}}}, nil, "reflective record component writes are unsupported")
}
func sourceReflectionRecordConstructors(scope *symbol.ClassScope, ctx Ctx) ast.Expr {
	constructors := &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("ConstructorDescriptor", ctx)}}
	var parameters []ast.Expr
	for _, field := range scope.Fields {
		if sourceReflectionRecordComponent(scope, field) == nil {
			continue
		}
		id, known := javaTypeDescriptorExpr(qualifyDeclaredReferenceType(symbol.JavaType{Original: field.OriginalType, TypeParameterBindings: field.TypeParameterBindings}, classScopeCtx(scope, ctx)), classScopeCtx(scope, ctx))
		if !known {
			return constructors
		}
		parameters = append(parameters, id)
	}
	constructors.Elts = append(constructors.Elts, &ast.CompositeLit{Elts: []ast.Expr{metadataKey("Parameters", sourceReflectionTypeIDs(parameters, ctx)), metadataKey("Modifiers", reflectionInteger(sourceReflectionModifiers(scope.Class, scope)&7))}})
	return constructors
}

// Component annotation propagation depends on @Target and implicit accessor /
// canonical-parameter rules. Refuse that separate service instead of inventing
// absent annotation metadata. Explicit record-body method annotations work.
func sourceReflectionRecordMetadataUnsupported(scope *symbol.ClassScope, ctx Ctx) string {
	if !sourceUsesReflection() || !sourceReflectionRecord(scope) {
		return ""
	}
	for _, field := range scope.Fields {
		if field.IsStatic {
			return "record reflection static state"
		}
	}
	for _, member := range nodeutil.NamedChildrenOf(scope.Class.DeclarationNode.ChildByFieldName("body")) {
		if member.Type() == "static_initializer" {
			return "record reflection static state"
		}
	}
	file := findFileScopeForClassScope(scope, ctx)
	if file == nil {
		return "record component annotation propagation"
	}
	for _, component := range nodeutil.NamedChildrenOf(scope.Class.DeclarationNode.ChildByFieldName("parameters")) {
		if component.Type() != "formal_parameter" {
			return "record component annotation propagation"
		}
		for _, child := range nodeutil.NamedChildrenOf(component) {
			if child.Type() == "modifiers" {
				for _, modifier := range nodeutil.NamedChildrenOf(child) {
					if modifier.Type() == "annotation" || modifier.Type() == "marker_annotation" {
						return "record component annotation propagation"
					}
				}
			}
		}
	}
	return ""
}
