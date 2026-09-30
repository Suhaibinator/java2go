package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// Library Entry has no source symbol. Invocation applicability must nevertheless
// follow the declared Java edge, keeping its invariant arguments independent of
// the erased Go protocol shape used to implement the entry.
func sourceMapEntryAssignable(actual, expected string, candidateTypeParams []string, ctx Ctx) bool {
	if canonicalMapEntryOwner(expected, ctx) == "" {
		return false
	}
	arguments, found := mapEntryTypeArguments(symbol.JavaType{Original: actual}, ctx, map[*symbol.ClassScope]bool{}, map[typeParameterIdentityKey]bool{})
	if !found {
		return false
	}
	_, expectedArguments := parseJavaTypeString(expected)
	return javaGenericArgumentsApplicable(arguments, expectedArguments, candidateTypeParams)
}

// Resolve each ancestor in its declaring file and substitute the actual owner
// arguments before following the next edge. A shared physical representation
// does not certify a Java nominal conversion.
func mapEntryTypeArguments(typ symbol.JavaType, ctx Ctx, scopes map[*symbol.ClassScope]bool, binders map[typeParameterIdentityKey]bool) ([]string, bool) {
	base, rank := javaArrayTypeParts(typ.Original)
	if rank != 0 {
		return nil, false
	}
	if binding, found := resolveReferenceTypeParameter(typ, ctx); found {
		identity := identityKeyForTypeParameter(binding.parameter)
		if binders[identity] {
			return nil, false
		}
		binders[identity] = true
		defer delete(binders, identity)
		for _, bound := range binding.parameter.Bounds {
			if arguments, ok := mapEntryTypeArguments(bound, binding.context, scopes, binders); ok {
				return arguments, true
			}
		}
		return nil, false
	}
	base, arguments := parseJavaTypeString(base)
	if canonicalMapEntryOwner(base, ctx) != "" {
		return arguments, len(arguments) == 0 || len(arguments) == 2
	}
	owner := resolveClassScopeByQualifiedName(ctx, base)
	if owner == nil || scopes[owner] {
		return nil, false
	}
	scopes[owner] = true
	defer delete(scopes, owner)
	declaring := classScopeCtx(owner, ctx)
	bindings := map[string]string{}
	for index, parameter := range owner.TypeParameters {
		argument := rawTypeParameterErasure(parameter, owner.TypeParameters)
		if index < len(arguments) {
			argument = arguments[index]
		}
		bindings[parameter.Name] = argument
		bindings[parameter.EmittedName()] = argument
	}
	for _, parent := range append(append([]string(nil), owner.ImplementedInterfaces...), owner.Superclass) {
		parent = substituteJavaTypeParams(qualifyJavaTypeInDeclaringContext(parent, owner), bindings)
		if entries, ok := mapEntryTypeArguments(symbol.JavaType{Original: parent}, declaring, scopes, binders); ok {
			if len(arguments) == 0 && len(owner.TypeParameters) != 0 {
				return nil, true // Raw owners expose erased generic supertypes.
			}
			return entries, true
		}
	}
	return nil, false
}
