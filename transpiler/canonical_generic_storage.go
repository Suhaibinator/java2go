package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// Physical canonical structs must not recursively instantiate their generic
// public aliases. Source declarations prove the target; pointer syntax only
// limits where this storage rewrite is representable.
func canonicalGenericStorageFields(fields *ast.FieldList, ctx Ctx) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		if physical, ok := canonicalGenericStorageFieldType(field, ctx); ok {
			field.Type = physical
		}
	}
}

func canonicalGenericStorageFieldType(field *ast.Field, ctx Ctx) (ast.Expr, bool) {
	owner := ctx.currentClass
	if field == nil || len(field.Names) != 1 || owner == nil || owner.Class == nil || owner.IsInterface || !canonicalGenericClass(owner, ctx) {
		return nil, false
	}
	declaring, owned := threadSourceDeclarationCtx(owner, nil, ctx)
	if !owned || declaring.currentFile != ctx.currentFile {
		return nil, false
	}
	var definition *symbol.Definition
	for _, candidate := range owner.Fields {
		if candidate != nil && candidate.Name == field.Names[0].Name {
			if definition != nil {
				return nil, false
			}
			definition = candidate
		}
	}
	var target *symbol.ClassScope
	if definition != nil {
		if definition.IsStatic || definition.DeclarationNode == nil || !threadSourceNodeWithin(definition.DeclarationNode, owner.Class.DeclarationNode) {
			return nil, false
		}
		origin := declaredJavaTypeOrigin(definitionDeclaredJavaType(definition), declaring)
		if !completeJavaTypeOrigin(origin) || origin.arrayRank != 0 || origin.parameter != nil || origin.nominalScope == nil {
			return nil, false
		}
		target = origin.nominalScope
	} else {
		// This field is synthesized from the actual enclosing declaration,
		// independently of a coincidentally similar Go type or source name.
		target = owner.Enclosing
		if !owner.IsInner || target == nil || target == owner || target.Class == nil || field.Names[0].Name != owner.EnclosingFieldName() || findFileScopeForClassScope(target, ctx) != ctx.currentFile || !threadSourceNodeWithin(owner.Class.DeclarationNode, target.Class.DeclarationNode) {
			return nil, false
		}
	}
	if target.Class == nil || target.Class.DeclarationNode == nil || target.IsInterface || !canonicalGenericClass(target, declaring) {
		return nil, false
	}
	if _, pointer := field.Type.(*ast.StarExpr); !pointer {
		return nil, false
	}
	name := availableCanonicalGenericName(target.Class.Name + "Java2goErased")
	return &ast.StarExpr{X: qualifiedNameExpr(name, findJavaPackageForClassScope(target), ctx)}, true
}
