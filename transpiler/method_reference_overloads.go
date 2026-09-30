package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
)

// Method references have SAM parameter types instead of invocation argument
// trees. Keep their applicability phases identical to ordinary method calls.
func methodReferenceConversionCost(actual, expected string, loose bool, ctx Ctx) (int, bool, bool) {
	if javaInferenceSameType(actual, expected, ctx) {
		return 0, true, true
	}
	actualPrimitive, actualIsPrimitive := javaPrimitiveType(actual)
	expectedPrimitive, expectedIsPrimitive := javaPrimitiveType(expected)
	if actualIsPrimitive && expectedIsPrimitive {
		distance, ok := javaPrimitiveWideningDistance(actualPrimitive, expectedPrimitive)
		return distance, false, ok
	}
	if !actualIsPrimitive && !expectedIsPrimitive {
		if applicable, _ := javaParameterAtLeastAsSpecific(actual, expected, ctx); applicable {
			return 16, false, true
		}
		if javaInferenceTypeAssignable(actual, expected, ctx) {
			return 24, false, true
		}
	}
	if !loose {
		return 0, false, false
	}
	if actualIsPrimitive && !expectedIsPrimitive {
		if builtinJavaReferenceAssignable("java.lang."+ternaryBoxedJavaType(actualPrimitive), expected, ctx) {
			return 8, false, true
		}
	}
	if primitive, boxed := javaUnboxingPrimitive(actual, ctx); boxed && expectedIsPrimitive {
		if primitive == expectedPrimitive {
			return 8, false, true
		}
		if distance, ok := javaPrimitiveWideningDistance(primitive, expectedPrimitive); ok {
			return 8 + distance, false, true
		}
	}
	return 0, false, false
}

func resolveMethodReferenceOverload(start *symbol.ClassScope, target *invocationTargetInfo, name string, actualTypes []string, static, constructor bool, ctx Ctx) *methodResolution {
	if start == nil {
		return nil
	}
	var best *methodResolution
	var bestScore methodCandidateScore
	seen := map[*symbol.ClassScope]bool{}
	var visit func(*symbol.ClassScope)
	visit = func(owner *symbol.ClassScope) {
		if owner == nil || seen[owner] {
			return
		}
		seen[owner] = true
		for _, definition := range owner.Methods {
			if definition == nil || definition.Constructor != constructor ||
				!constructor && (definition.IsStatic != static || definition.OriginalName != name) {
				continue
			}
			candidate := &methodResolution{def: definition, owner: owner, receiverScope: start}
			var ownerArguments []string
			if target != nil {
				ownerArguments = invocationOwnerTypeArguments(target, candidate, ctx)
			}
			expectedTypes := instantiatedMethodParameterTypes(candidate, ownerArguments)
			_, resultType := methodReferenceJavaSignature(ctx)
			bindings := methodReferenceInferredBindings(definition, actualTypes, resultType, ctx)
			for index := range expectedTypes {
				expectedTypes[index] = substituteJavaTypeParams(expectedTypes[index], bindings)
			}
			variadic := len(definition.Parameters) > 0 && executionParameterIsVariadic(definition, len(definition.Parameters)-1)
			for phase := 0; phase < 3; phase++ {
				if phase < 2 && len(actualTypes) != len(expectedTypes) {
					continue
				}
				if phase == 2 && (!variadic || len(actualTypes) < len(expectedTypes)-1) {
					continue
				}
				score := methodCandidateScore{phase: phase}
				applicable := true
				for index, actual := range actualTypes {
					formalIndex := index
					if phase == 2 && formalIndex >= len(expectedTypes)-1 {
						formalIndex = len(expectedTypes) - 1
					}
					if formalIndex < 0 || formalIndex >= len(expectedTypes) {
						applicable = false
						break
					}
					expected := expectedTypes[formalIndex]
					if phase < 2 && variadic && formalIndex == len(expectedTypes)-1 {
						expected += "[]"
					}
					cost, exact, ok := methodReferenceConversionCost(actual, expected, phase > 0, ctx)
					if !ok {
						applicable = false
						break
					}
					score.totalCost += cost
					if exact {
						score.exactCount++
					}
				}
				if !applicable {
					continue
				}
				candidate.expandVarargsArray = variadic && phase < 2
				if best == nil || methodCandidateScoreBetter(score, bestScore) ||
					score.phase == bestScore.phase && score.totalCost == bestScore.totalCost && score.exactCount == bestScore.exactCount && methodResolutionMoreSpecific(candidate, best, ctx) {
					best, bestScore = candidate, score
				}
				break
			}
		}
		if constructor {
			return
		}
		visit(resolveSuperclassScopeInDeclaringContext(ctx, owner))
		for _, implemented := range owner.ImplementedInterfaces {
			base, _ := parseJavaTypeString(implemented)
			// Static interface methods are not inherited.
			if !static {
				visit(resolveClassScopeByQualifiedName(classScopeCtx(owner, ctx), base))
			}
		}
	}
	visit(start)
	return best
}
