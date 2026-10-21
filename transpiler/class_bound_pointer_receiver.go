package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// A Go type parameter does not expose fields, even with an exact *Source
// constraint. Convert to that already-emitted pointer before source dispatch.
// Method binders use their existing projection witnesses instead.
func classBoundConcretePointerReceiverExpr(value ast.Expr, javaType string, expectedScope *symbol.ClassScope, ctx Ctx) (ast.Expr, bool) {
	declaration := visibleTypeParameterDeclarationForJavaType(javaType, ctx)
	if value == nil || declaration == nil || expectedScope == nil {
		return nil, false
	}
	if ctx.localScope != nil {
		for _, parameter := range ctx.localScope.TypeParameters {
			if parameter.Declaration == declaration {
				return nil, false
			}
		}
	}
	if ctx.dependentTypeWitnesses != nil && ctx.dependentTypeWitnesses.hasSource(declaration) {
		return nil, false
	}
	for owner := ctx.currentClass; owner != nil; owner = owner.Enclosing {
		for index, parameter := range owner.TypeParameters {
			if parameter.Declaration != declaration {
				continue
			}
			if len(parameter.Bounds) != 1 || canonicalGenericClass(owner, ctx) {
				return nil, false
			}
			declaring := classHeaderTypeCtx(owner, ctx)
			bound := parameter.Bounds[0]
			if _, dependent := resolveReferenceTypeParameter(bound, declaring); dependent {
				return nil, false
			}
			qualified := qualifyDeclaredReferenceType(bound, declaring)
			base, rank := javaArrayTypeParts(strings.TrimSpace(qualified))
			base, arguments := parseJavaTypeString(base)
			if rank != 0 || len(arguments) != 0 {
				return nil, false
			}
			boundScope := resolveClassScopeByQualifiedName(declaring, base)
			if boundScope == nil || boundScope != expectedScope || boundScope.IsInterface || abstractClassUsesInterfaceView(boundScope) {
				return nil, false
			}
			fields := makeTypeParamFieldsInContext(owner.TypeParameters, declaring)
			if index >= len(fields) {
				return nil, false
			}
			pointer, ok := fields[index].Type.(*ast.StarExpr)
			if !ok {
				return nil, false
			}
			return &ast.CallExpr{Fun: pointer, Args: []ast.Expr{value}}, true
		}
	}
	return nil, false
}
