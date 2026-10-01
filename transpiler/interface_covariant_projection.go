package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// A source interface covariant override shares its public selector with the
// wider ancestor. Keep that public descriptor compatible with Go embedding;
// its execution companion retains the exact source result and selector.
func interfaceCovariantPublicResultType(owner *symbol.ClassScope, method *symbol.Definition, declared ast.Expr, ctx Ctx) ast.Expr {
	if owner == nil || !owner.IsInterface {
		return declared
	}
	selection, ok := specializedAncestorCovariantBridge(owner, method, classScopeCtx(owner, ctx))
	if !ok {
		return declared
	}
	declarationCtx := classScopeCtx(selection.family.owner, ctx)
	result := javaTypeStringToGoTypeExpr(selection.family.erasedResult, inScopeTypeParameters(declarationCtx), declarationCtx)
	return abstractClassToInterface(result, selection.family.erasedResult, declarationCtx)
}

// Handwritten implementations may only provide the wider public descriptor.
// Restore the declared Java result with the same nominal, null-preserving cast
// used for erased source results before returning it to a narrow caller.
func projectInterfaceCovariantPublicResult(call ast.Expr, resolution *methodResolution, ctx Ctx) ast.Expr {
	if resolution == nil || resolution.owner == nil || !resolution.owner.IsInterface {
		return call
	}
	selection, ok := specializedAncestorCovariantBridge(resolution.owner, resolution.def, classScopeCtx(resolution.owner, ctx))
	if !ok {
		return call
	}
	return projectDirectOwnerErasedView(call, qualifyJavaTypeInDeclaringContext(resolution.def.OriginalType, resolution.owner), qualifyJavaTypeInDeclaringContext(selection.family.erasedResult, selection.family.owner), ctx)
}
