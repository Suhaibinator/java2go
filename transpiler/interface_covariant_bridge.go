package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// A covariant source override needs a bridge to its inherited descriptor.
// Interfaces retain their instantiated type arguments; source superclass
// methods that do not use class binders retain their ordinary wider result.
func specializedAncestorCovariantBridge(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) (directOwnerSpecializedOverrideBridgeSelection, bool) {
	var selected directOwnerSpecializedOverrideBridgeSelection
	if owner == nil || method == nil || method.IsPrivate || method.IsStatic || method.Constructor || len(method.TypeParameters) != 0 {
		return selected, false
	}
	queue := []*symbol.ClassScope{owner}
	seen := map[*symbol.ClassScope]bool{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == nil || seen[current] {
			continue
		}
		seen[current] = true
		queue = append(queue, resolveImplementedInterfaceScopesInDeclaringContext(ctx, current)...)
		if !current.IsInterface {
			queue = append(queue, resolveSuperclassScopeInDeclaringContext(ctx, current))
			if current == owner {
				continue
			}
		}
		args := mapClassTypeArgumentStringsToAncestor(owner, owner.GoTypeParameterNames(), current, ctx)
		for _, candidate := range current.Methods {
			if candidate.IsStatic || candidate.IsPrivate || candidate.OriginalName != method.OriginalName || candidate.Name != method.Name || len(candidate.Parameters) != len(method.Parameters) || len(candidate.TypeParameters) != 0 {
				continue
			}
			parameters, result := mappedOverrideSourceSignature(current, candidate, args)
			for index, parameter := range parameters {
				declaring := overrideBridgeMappedTypeOwner(current, owner, candidate.Parameters[index])
				parameters[index] = qualifyJavaTypeInDeclaringContext(parameter, declaring)
			}
			result = qualifyJavaTypeInDeclaringContext(result, overrideBridgeMappedTypeOwner(current, owner, candidate))
			if !current.IsInterface {
				// Reuse Java override admission, including declaration-package
				// access. A same-name inaccessible ancestor is a shadow, not
				// a member of this virtual family.
				admitted := false
				for _, override := range directDeclaredOverrides(current, owner, candidate, parameters, result, ctx) {
					if override == method {
						admitted = true
						break
					}
				}
				if !admitted {
					continue
				}
			}
			actual := qualifyJavaTypeInDeclaringContext(method.OriginalType, owner)
			if overrideBridgeJavaTypesIdentical(actual, owner, result, current, ctx) {
				continue
			}
			matches := true
			for index, param := range parameters {
				if !overrideBridgeJavaTypesIdentical(method.Parameters[index].OriginalType, owner, param, current, ctx) {
					matches = false
					break
				}
			}
			if !matches || !overrideBridgeResultCompatible(actual, owner, result, current, ctx) {
				continue
			}
			if !overrideBridgePlainResultWideningSupported(result, current, ctx) && !overrideBridgeConcreteResultWideningSupported(actual, owner, result, current, ctx) {
				continue
			}
			bridge := directOwnerOverrideBridgePlan{owner: owner, method: method, ancestorArgs: args, result: directOwnerOverrideBridgeResultPlan{erasedJavaType: result, sourceJavaType: result, overrideJavaType: actual, requiresWidening: true}}
			for _, param := range parameters {
				bridge.parameters = append(bridge.parameters, directOwnerOverrideBridgeParameterPlan{erasedJavaType: param, sourceJavaType: param, overrideJavaType: param})
			}
			family := &directOwnerOverrideBridgeFamilyPlan{owner: current, method: candidate, erasedParameters: parameters, erasedResult: result}
			if selected.family != nil && overrideBridgeFamilyDescriptorKey(selected.family, ctx) != overrideBridgeFamilyDescriptorKey(family, ctx) {
				return directOwnerSpecializedOverrideBridgeSelection{}, false
			}
			selected = directOwnerSpecializedOverrideBridgeSelection{family: family, bridge: bridge}
		}
	}
	return selected, selected.family != nil
}
