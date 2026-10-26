package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
)

func sourceNativeVirtualResultErasure(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) (string, bool) {
	plan := sourceNativeVirtualBodyForMethod(owner, method, ctx)
	if plan == nil || !sourceNativeVirtualScalarDefinition(method, plan.declarations) {
		return "", false
	}
	return "java.lang.Object", true
}
func sourceNativeVirtualParameterErasure(owner *symbol.ClassScope, method *symbol.Definition, index int, ctx Ctx) (string, bool) {
	if method == nil || index < 0 || index >= len(method.Parameters) {
		return "", false
	}
	plan := sourceNativeVirtualBodyForMethod(owner, method, ctx)
	if plan == nil || !sourceNativeVirtualScalarDefinition(method.Parameters[index], plan.declarations) {
		return "", false
	}
	return "java.lang.Object", true
}
func sourceNativeVirtualHasParameterABI(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) bool {
	plan := sourceNativeVirtualBodyForMethod(owner, method, ctx)
	if plan == nil {
		return false
	}
	for _, parameter := range method.Parameters {
		if sourceNativeVirtualScalarDefinition(parameter, plan.declarations) {
			return true
		}
	}
	return false
}
func buildSourceNativeVirtualBodyDecls(declaration *ast.FuncDecl, sourceParams, sourceResults *ast.FieldList, executionName string, ctx Ctx) ([]ast.Decl, bool) {
	plan := ctx.sourceNativeVirtualBody
	if plan == nil || plan.owner != ctx.currentClass || plan.method != ctx.localScope {
		return nil, false
	}
	// Reuse only the emitter. This adapter is not added to the source-family or
	// field-layout eligibility planners, and never changes a receiver's Go type.
	family := &directOwnerOverrideBridgeFamilyPlan{owner: plan.owner, method: plan.method, erasedParameters: plan.erasedParameters, erasedResult: plan.erasedResult, requiresErasedView: true}
	declarations := buildDirectOwnerErasedFamilyMethodDecls(declaration, sourceParams, sourceResults, executionName, family, ctx)
	if selection, ok := sourceAbstractListSpecializedOverride(plan.owner, plan.method, ctx); ok {
		if bridge := buildDirectOwnerSpecializedOverrideBridgeDecl(declaration, executionName, selection, ctx); bridge != nil {
			declarations = append(declarations, bridge)
		}
	}
	return declarations, true
}
