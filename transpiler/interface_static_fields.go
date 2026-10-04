package transpiler

import (
	"go/ast"
	"go/token"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Interface fields have the same storage and active-use initialization rules
// as static class fields. Their implicit public/static/final modifiers are
// resolved by symbol parsing; compile-time constants do not trigger the lazy
// initializer, while other values are assigned in Java declaration order.
func interfaceStaticFieldDeclarations(body *sitter.Node, source []byte, ctx Ctx) []ast.Decl {
	if body == nil || ctx.currentClass == nil || len(ctx.currentClass.Fields) == 0 {
		return nil
	}
	resolveCompileTimeConstantsForClass(ctx.currentClass, source, ctx)
	variables := &ast.GenDecl{Tok: token.VAR}
	for _, field := range ctx.currentClass.Fields {
		if !field.IsStatic {
			continue
		}
		typ := abstractClassToInterface(javaTypeStringToGoTypeExpr(field.OriginalType, ctx.currentClass.TypeParameterNames(), ctx), field.OriginalType, ctx)
		spec := &ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent(field.Name)}, Type: typ}
		if field.IsCompileTimeConstant {
			if declarator := declaratorForField(field, source); declarator != nil {
				if valueNode := declarator.ChildByFieldName("value"); valueNode != nil {
					valueCtx := ctx.Clone()
					valueCtx.localScope = &symbol.Definition{IsStatic: true}
					valueCtx.expectedType = field.OriginalType
					valueCtx.expectedTypeRoot = valueNode
					value := ParseExpr(valueNode, source, valueCtx)
					spec.Values = []ast.Expr{coerceArgumentToExpectedType(value, valueNode, field.OriginalType, valueCtx, source)}
				}
			}
		}
		variables.Specs = append(variables.Specs, spec)
	}
	var declarations []ast.Decl
	if len(variables.Specs) > 0 {
		declarations = append(declarations, variables)
		declarations = append(declarations, staticFieldStorageHelperDecls(variables, ctx)...)
	}
	declarations = append(declarations, buildLazyClassInitializationDecls(body, source, ctx)...)
	return declarations
}
