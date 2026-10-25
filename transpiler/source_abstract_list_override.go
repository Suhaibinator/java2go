package transpiler

import (
	"fmt"
	"github.com/NickyBoy89/java2go/symbol"
)

// The platform-parent body carrier has an erased hidden descriptor while its
// public Go wrapper retains the source instantiation. A concrete source override
// needs the same checked javac-style bridge as an ordinary generic ancestor.
// This proves that one native-backed family; it does not enable general native
// families in the ordinary field/container representation planner.
func sourceAbstractListSpecializedOverride(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) (directOwnerSpecializedOverrideBridgeSelection, bool) {
	if method == nil || method.IsPrivate || method.IsStatic || method.Constructor || method.RuntimeDefault || len(method.TypeParameters) != 0 || sourceAbstractListRoot(owner, ctx) == nil {
		return directOwnerSpecializedOverrideBridgeSelection{}, false
	}
	declaring, owned := sourceAbstractListDeclaration(owner, ctx)
	if !owned || !sourceAbstractListActualMethodOwned(method, owner, declaring) {
		return directOwnerSpecializedOverrideBridgeSelection{}, false
	}
	seen := map[*symbol.ClassScope]bool{owner: true}
	for parent := resolveSuperclassScopeInDeclaringContext(ctx, owner); parent != nil && !seen[parent]; parent = resolveSuperclassScopeInDeclaringContext(ctx, parent) {
		seen[parent] = true
		if sourceAbstractListRoot(parent, ctx) == nil {
			return directOwnerSpecializedOverrideBridgeSelection{}, false
		}
		for _, inherited := range parent.Methods {
			if inherited == nil || inherited.IsPrivate || inherited.IsStatic || inherited.Constructor || inherited.OriginalName != method.OriginalName || len(inherited.Parameters) != len(method.Parameters) {
				continue
			}
			native := sourceAbstractListVirtualBody(parent, inherited, ctx)
			if !sourceAbstractListErasedSignature(native) {
				continue
			}
			overrides := directOwnerOverrideBridgeMethodDeclaresOverride(parent, inherited, owner, method, ctx)
			if inherited.RuntimeDefault {
				overrides = inheritedOverloadOverrides(owner, method, parent, inherited)
			}
			if !overrides {
				continue
			}
			family := &directOwnerOverrideBridgeFamilyPlan{owner: parent, method: inherited, erasedParameters: append([]string(nil), native.erasedParameters...), erasedResult: native.erasedResult, requiresErasedView: true}
			bridge := directOwnerOverrideBridgePlan{owner: owner, method: method, ancestorArgs: mapClassTypeArgumentStringsToAncestor(owner, owner.GoTypeParameterNames(), parent, ctx), sourceNativeExactExecutionName: collisionSafeExecutionIdentifier(method.Name+"Java2goAbstractListExactFor"+fmt.Sprintf("%X", []byte(sourceClassRuntimeTypeID(owner, ctx))), owner)}
			ownNative := sourceAbstractListVirtualBody(owner, method, ctx)
			for index, parameter := range method.Parameters {
				exact := qualifyJavaTypeInDeclaringContext(definitionParameterJavaSignatureType(method, index), owner)
				if ownNative != nil && sourceNativeVirtualScalarDefinition(parameter, ownNative.declarations) {
					exact = "java.lang.Object"
				}
				erased := family.erasedParameters[index]
				bridge.parameters = append(bridge.parameters, directOwnerOverrideBridgeParameterPlan{erasedJavaType: erased, sourceJavaType: sourceAbstractListInstantiatedMethodType(inherited.Parameters[index].OriginalType, parent, owner, ctx), overrideJavaType: exact, requiresCast: !javaInferenceSameType(erased, exact, declaring)})
			}
			exactResult := qualifyJavaTypeInDeclaringContext(method.OriginalType, owner)
			if ownNative != nil && sourceNativeVirtualScalarDefinition(method, ownNative.declarations) {
				exactResult = "java.lang.Object"
			}
			bridge.result = directOwnerOverrideBridgeResultPlan{erasedJavaType: family.erasedResult, sourceJavaType: sourceAbstractListInstantiatedMethodType(inherited.OriginalType, parent, owner, ctx), overrideJavaType: exactResult, requiresWidening: !javaInferenceSameType(family.erasedResult, exactResult, declaring)}
			return directOwnerSpecializedOverrideBridgeSelection{family: family, bridge: bridge}, true
		}
	}
	return directOwnerSpecializedOverrideBridgeSelection{}, false
}
