package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"strings"
)

// Proposed only: the caller has already proved erased source storage/result,
// an exact consuming target and an erasure which cannot satisfy that target.
func projectDirectOwnerErasedExpectedArray(expr ast.Expr, target string, ctx Ctx) (ast.Expr, bool) {
	target = strings.TrimSpace(target)
	_, rank := javaArrayTypeParts(target)
	if expr == nil || rank < 1 {
		return nil, false
	}
	origin := declaredJavaTypeOrigin(symbol.JavaType{Original: target, TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx)
	if origin.arrayRank != rank || !closedDeclaredAncestryOrigin(origin) || len(origin.arguments) != 0 {
		return nil, false
	}
	descriptor, known := declaredOriginDescriptor(origin, ctx, map[*symbol.TypeParamDeclaration]bool{})
	if !known {
		return nil, false
	}
	physical, primitive := primitiveArrayTypeExpr(target, ctx)
	if !primitive {
		physical = reifiedReferenceArrayTypeExpr(ctx)
	}
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{physical}, []ast.Expr{expr, descriptor}), true
}
