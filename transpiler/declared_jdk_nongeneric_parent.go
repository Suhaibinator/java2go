package transpiler

// declaredJDKNonGenericParentOriginAssignable supplies positive evidence only.
// Generic target arguments retain the existing substitution/variance machinery;
// unavailable declarations, arguments or compatibility proofs fall through.
func declaredJDKNonGenericParentOriginAssignable(actual, formal inferredJavaTypeOrigin, ctx Ctx) bool {
	if actual.nominalScope != nil || formal.nominalScope != nil || actual.parameter != nil || formal.parameter != nil || actual.arrayRank != 0 || formal.arrayRank != 0 || actual.wildcard || formal.wildcard {
		return false
	}
	target, ok := certifiedJDK21DeclarationBounds(formal.externalName)
	if !ok || len(target.parameters) != 0 || len(formal.arguments) != 0 {
		return false
	}
	supplied, ok := declaredArgumentProof(actual, actual.javaType, ctx, 0)
	if !ok || !declaredJDKParentArgumentsValid(supplied, ctx, 0) {
		return false
	}
	requested, ok := declaredArgumentProof(formal, formal.javaType, ctx, 0)
	if !ok || !declaredJDKParentArgumentsValid(requested, ctx, 0) {
		return false
	}
	return jdk21DeclaredBoundSubtype(actual, formal) == declaredBoundYes
}

// Validate proper supplied arguments against their actual declaration bounds.
// A wildcard uses the existing effective-upper reduction. Multiple remaining
// conjuncts need a separate compatibility proof, so remain explicitly unknown.
func declaredJDKParentArgumentsValid(view declaredProofArgument, ctx Ctx, depth int) bool {
	if depth > 64 || view.variance != "" {
		return false
	}
	for slot, argument := range view.arguments {
		if argument.variance != "" {
			upper, ok := declaredEffectiveWildcardUpper(view, slot, ctx, depth+1)
			if !ok || len(upper) != 1 || !declaredJDKParentArgumentsValid(upper[0], ctx, depth+1) {
				return false
			}
			continue
		}
		if !declaredJDKParentArgumentsValid(argument, ctx, depth+1) {
			return false
		}
		bounds, ok := declaredGenericFormalBounds(view, slot, ctx, depth+1)
		if !ok {
			return false
		}
		for _, bound := range bounds {
			if declaredProofBoundSubtype(argument, bound, ctx, depth+1) != declaredBoundYes {
				return false
			}
		}
	}
	return true
}
