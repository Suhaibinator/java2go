package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// directOwnerOverrideBridgeFamilyPlan is the declaration-identity-aware plan
// for one generic method family that cannot use one invariant Go signature.
//
// Java gives the ancestor method an erased descriptor, while a concrete
// specialization may declare a narrower source override. javac keeps the
// narrow method and adds a synthetic bridge with the ancestor descriptor. Go
// cannot overload those two methods, so later lowering consumes this plan as
// two distinct generated selectors: a narrow implementation selector and one
// stable erased family selector.
//
// Planning is deliberately separate from emission. In particular, a family is
// returned only when the complete connected class-parameter representation can
// move atomically to erased storage and callable bodies. That prevents a bridge
// for one method from making a sibling field or method ABI inconsistent.
type directOwnerOverrideBridgeFamilyPlan struct {
	owner              *symbol.ClassScope
	method             *symbol.Definition
	erasedParameters   []string
	erasedResult       string
	requiresErasedView bool
	overrides          []directOwnerOverrideBridgePlan
}

type directOwnerOverrideBridgePlan struct {
	owner        *symbol.ClassScope
	method       *symbol.Definition
	ancestorArgs []string
	parameters   []directOwnerOverrideBridgeParameterPlan
	result       directOwnerOverrideBridgeResultPlan
}

type directOwnerOverrideBridgeParameterPlan struct {
	erasedJavaType   string
	sourceJavaType   string
	overrideJavaType string
	requiresCast     bool
}

type directOwnerOverrideBridgeResultPlan struct {
	erasedJavaType   string
	sourceJavaType   string
	overrideJavaType string
	requiresWidening bool
}

// planDirectOwnerCallableOverrideBridgeFamily is the integration boundary for
// the future physical-ABI and declaration passes. The unchecked method-family
// plan first proves every override bridge locally; the representation audit
// then proves that all uses of every connected class type parameter can migrate
// together. No partial family is ever exposed to code generation.
func planDirectOwnerCallableOverrideBridgeFamily(
	owner *symbol.ClassScope,
	method *symbol.Definition,
	ctx Ctx,
) (*directOwnerOverrideBridgeFamilyPlan, bool) {
	plan, ok := planDirectOwnerCallableOverrideBridgeFamilyUnchecked(owner, method, ctx)
	if !ok || plan == nil || !plan.requiresErasedView {
		return nil, false
	}
	if !directOwnerOverrideBridgeRepresentationSupported(plan, ctx) {
		return nil, false
	}
	return plan, true
}

// planDirectOwnerCallableOverrideBridgeFamilyUnchecked validates one method
// family without recursively auditing sibling methods. It is used by the
// atomic representation audit to classify each sibling independently.
func planDirectOwnerCallableOverrideBridgeFamilyUnchecked(
	owner *symbol.ClassScope,
	method *symbol.Definition,
	ctx Ctx,
) (*directOwnerOverrideBridgeFamilyPlan, bool) {
	if !ordinaryConcreteCallableOwner(owner) || !ordinarySourceMethod(owner, method) ||
		len(owner.TypeParameters) == 0 || len(method.TypeParameters) != 0 ||
		executionParameterIsVariadic(method, len(method.Parameters)-1) ||
		classHasUnmodeledCallableSubclass(owner, ctx) {
		return nil, false
	}

	erasedParameters := make([]string, len(method.Parameters))
	changed := false
	for index, parameter := range method.Parameters {
		erased, direct, ok := directOwnerOverrideBridgeErasedType(owner, parameter, ctx)
		if !ok {
			return nil, false
		}
		erasedParameters[index] = erased
		changed = changed || direct
	}

	erasedResult := "void"
	if !javaMethodResultIsVoid(method) {
		var direct bool
		var ok bool
		erasedResult, direct, ok = directOwnerOverrideBridgeErasedType(owner, method, ctx)
		if !ok {
			return nil, false
		}
		changed = changed || direct
	}
	if !changed {
		return nil, false
	}

	plan := &directOwnerOverrideBridgeFamilyPlan{
		owner:            owner,
		method:           method,
		erasedParameters: erasedParameters,
		erasedResult:     erasedResult,
	}

	valid := true
	visitAllClassScopes(func(descendant *symbol.ClassScope) bool {
		if descendant == nil || descendant == owner || !classScopeDescendsFrom(descendant, owner, ctx) {
			return false
		}
		if !ordinaryConcreteCallableOwner(descendant) {
			valid = false
			return true
		}

		ancestorArgs := mapClassTypeArgumentStringsToAncestor(
			descendant,
			descendant.GoTypeParameterNames(),
			owner,
			ctx,
		)
		if len(ancestorArgs) != len(owner.TypeParameters) {
			valid = false
			return true
		}
		if !descendantPreservesCallableOwnerParameters(descendant, owner, method, ctx) {
			plan.requiresErasedView = true
		}

		mappedParameters, mappedResult := mappedOverrideSourceSignature(owner, method, ancestorArgs)
		matching := directDeclaredOverrides(descendant, method, mappedParameters, mappedResult, ctx)
		if len(matching) > 1 {
			// A legal Java declaration cannot contain two override-equivalent methods,
			// but conservatively reject malformed or incompletely-resolved symbols.
			valid = false
			return true
		}
		if len(matching) == 0 {
			// A concrete specialization without an override inherits the ancestor's
			// erased entry and needs no synthetic bridge of its own.
			return false
		}

		overridePlan, bridgeRequired, bridgeOK := planDirectOwnerSpecializedOverride(
			owner,
			method,
			descendant,
			matching[0],
			ancestorArgs,
			mappedParameters,
			mappedResult,
			erasedParameters,
			erasedResult,
			ctx,
		)
		if !bridgeOK {
			valid = false
			return true
		}
		if bridgeRequired {
			plan.requiresErasedView = true
			plan.overrides = append(plan.overrides, overridePlan)
		}
		return false
	})
	if !valid || !plan.requiresErasedView {
		return nil, false
	}
	return plan, true
}

func directOwnerOverrideBridgeErasedType(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	ctx Ctx,
) (javaType string, direct bool, ok bool) {
	if owner == nil || definition == nil {
		return "", false, false
	}
	if _, isDirect := directOwnerTypeParameterForDefinition(owner, definition); isDirect {
		erasure, interfaceErasure := directOwnerInterfaceErasure(owner, definition, classScopeCtx(owner, ctx))
		if !interfaceErasure {
			return "", false, false
		}
		return qualifyJavaTypeInDeclaringContext(erasure, owner), true, true
	}
	for _, parameter := range owner.TypeParameters {
		if parameter.Declaration != nil && definitionReferencesTypeParameterDeclaration(definition, parameter.Declaration) {
			// Arrays and nested parameterized uses require a separate canonical
			// representation plan; do not accidentally call them bridgeable here.
			return "", false, false
		}
	}
	return qualifyJavaTypeInDeclaringContext(definitionJavaType(definition), owner), false, true
}

func javaMethodResultIsVoid(method *symbol.Definition) bool {
	if method == nil {
		return true
	}
	result := strings.TrimSpace(method.OriginalType)
	return result == "" || result == "void"
}

func mappedOverrideSourceSignature(
	owner *symbol.ClassScope,
	method *symbol.Definition,
	ancestorArgs []string,
) ([]string, string) {
	bindings := make(map[string]string, len(owner.TypeParameters)*2)
	for index, parameter := range owner.TypeParameters {
		if index >= len(ancestorArgs) {
			continue
		}
		bindings[parameter.Name] = ancestorArgs[index]
		bindings[parameter.EmittedName()] = ancestorArgs[index]
	}
	parameters := make([]string, len(method.Parameters))
	for index := range method.Parameters {
		parameters[index] = substituteJavaTypeParams(definitionParameterJavaSignatureType(method, index), bindings)
	}
	result := "void"
	if !javaMethodResultIsVoid(method) {
		result = substituteJavaTypeParams(method.OriginalType, bindings)
	}
	return parameters, result
}

func directDeclaredOverrides(
	owner *symbol.ClassScope,
	ancestorMethod *symbol.Definition,
	mappedParameters []string,
	mappedResult string,
	ctx Ctx,
) []*symbol.Definition {
	if owner == nil || ancestorMethod == nil {
		return nil
	}
	var matches []*symbol.Definition
	for _, candidate := range owner.Methods {
		if candidate == nil || candidate.Constructor || candidate.IsStatic || candidate.IsPrivate ||
			candidate.RequiresHelper || candidate.DeclarationNode == nil ||
			candidate.DeclarationNode.Type() != "method_declaration" ||
			candidate.OriginalName != ancestorMethod.OriginalName ||
			len(candidate.Parameters) != len(mappedParameters) || len(candidate.TypeParameters) != 0 {
			continue
		}
		parametersMatch := true
		for index := range candidate.Parameters {
			if !overrideBridgeJavaTypesIdentical(mappedParameters[index], owner, definitionParameterJavaSignatureType(candidate, index), owner, ctx) {
				parametersMatch = false
				break
			}
		}
		if !parametersMatch || !overrideBridgeResultCompatible(candidate.OriginalType, owner, mappedResult, owner, ctx) {
			continue
		}
		matches = append(matches, candidate)
	}
	return matches
}

func planDirectOwnerSpecializedOverride(
	ancestorOwner *symbol.ClassScope,
	ancestorMethod *symbol.Definition,
	overrideOwner *symbol.ClassScope,
	overrideMethod *symbol.Definition,
	ancestorArgs []string,
	mappedParameters []string,
	mappedResult string,
	erasedParameters []string,
	erasedResult string,
	ctx Ctx,
) (directOwnerOverrideBridgePlan, bool, bool) {
	if ancestorOwner == nil || ancestorMethod == nil || overrideOwner == nil || overrideMethod == nil ||
		len(overrideMethod.Parameters) != len(erasedParameters) ||
		executionParameterIsVariadic(overrideMethod, len(overrideMethod.Parameters)-1) {
		return directOwnerOverrideBridgePlan{}, false, false
	}

	plan := directOwnerOverrideBridgePlan{
		owner:        overrideOwner,
		method:       overrideMethod,
		ancestorArgs: append([]string(nil), ancestorArgs...),
		parameters:   make([]directOwnerOverrideBridgeParameterPlan, len(erasedParameters)),
	}
	bridgeRequired := false
	for index := range overrideMethod.Parameters {
		overrideType := qualifyJavaTypeInDeclaringContext(definitionParameterJavaSignatureType(overrideMethod, index), overrideOwner)
		sourceType := qualifyJavaTypeInDeclaringContext(mappedParameters[index], overrideOwner)
		if !overrideBridgeJavaTypesIdentical(sourceType, overrideOwner, overrideType, overrideOwner, ctx) {
			return directOwnerOverrideBridgePlan{}, false, false
		}
		overrideDescriptor, _, descriptorOK := directOwnerOverrideBridgeErasedType(overrideOwner, overrideMethod.Parameters[index], ctx)
		if !descriptorOK {
			return directOwnerOverrideBridgePlan{}, false, false
		}
		requiresCast := !overrideBridgeJavaTypesIdentical(erasedParameters[index], ancestorOwner, overrideDescriptor, overrideOwner, ctx)
		if requiresCast && !overrideBridgeCastTargetSupported(overrideType) {
			return directOwnerOverrideBridgePlan{}, false, false
		}
		plan.parameters[index] = directOwnerOverrideBridgeParameterPlan{
			erasedJavaType:   erasedParameters[index],
			sourceJavaType:   sourceType,
			overrideJavaType: overrideType,
			requiresCast:     requiresCast,
		}
		bridgeRequired = bridgeRequired || requiresCast
	}

	overrideResult := "void"
	overrideResultDescriptor := "void"
	if !javaMethodResultIsVoid(overrideMethod) {
		overrideResult = qualifyJavaTypeInDeclaringContext(overrideMethod.OriginalType, overrideOwner)
		var descriptorOK bool
		overrideResultDescriptor, _, descriptorOK = directOwnerOverrideBridgeErasedType(overrideOwner, overrideMethod, ctx)
		if !descriptorOK {
			return directOwnerOverrideBridgePlan{}, false, false
		}
	}
	if !overrideBridgeResultCompatible(overrideResult, overrideOwner, mappedResult, overrideOwner, ctx) ||
		!overrideBridgeResultCompatible(overrideResultDescriptor, overrideOwner, erasedResult, ancestorOwner, ctx) {
		return directOwnerOverrideBridgePlan{}, false, false
	}
	resultDescriptorDiffers := !overrideBridgeJavaTypesIdentical(erasedResult, ancestorOwner, overrideResultDescriptor, overrideOwner, ctx)
	resultNeedsWidening := !overrideBridgeJavaTypesIdentical(erasedResult, ancestorOwner, overrideResult, overrideOwner, ctx)
	plan.result = directOwnerOverrideBridgeResultPlan{
		erasedJavaType:   erasedResult,
		sourceJavaType:   qualifyJavaTypeInDeclaringContext(mappedResult, overrideOwner),
		overrideJavaType: overrideResult,
		requiresWidening: resultNeedsWidening,
	}
	bridgeRequired = bridgeRequired || resultDescriptorDiffers
	return plan, bridgeRequired, true
}

func overrideBridgeCastTargetSupported(javaType string) bool {
	component, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	if rank != 0 || strings.TrimSpace(component) == "" {
		return false
	}
	_, primitive := javaPrimitiveType(component)
	return !primitive
}

func overrideBridgeJavaTypesIdentical(
	left string,
	leftOwner *symbol.ClassScope,
	right string,
	rightOwner *symbol.ClassScope,
	ctx Ctx,
) bool {
	left = qualifyJavaTypeInDeclaringContext(strings.TrimSpace(left), leftOwner)
	right = qualifyJavaTypeInDeclaringContext(strings.TrimSpace(right), rightOwner)
	leftCtx := classScopeCtx(leftOwner, ctx)
	rightCtx := classScopeCtx(rightOwner, ctx)
	return callablePhysicalTypeKey(left, leftCtx) == callablePhysicalTypeKey(right, rightCtx)
}

func overrideBridgeResultCompatible(
	actual string,
	actualOwner *symbol.ClassScope,
	expected string,
	expectedOwner *symbol.ClassScope,
	ctx Ctx,
) bool {
	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)
	if actual == "" {
		actual = "void"
	}
	if expected == "" {
		expected = "void"
	}
	if overrideBridgeJavaTypesIdentical(actual, actualOwner, expected, expectedOwner, ctx) {
		return true
	}
	if actual == "void" || expected == "void" {
		return false
	}
	actualComponent, actualRank := javaArrayTypeParts(actual)
	expectedComponent, expectedRank := javaArrayTypeParts(expected)
	if actualRank != expectedRank {
		return false
	}
	if _, primitive := javaPrimitiveType(actualComponent); primitive {
		return false
	}
	if _, primitive := javaPrimitiveType(expectedComponent); primitive {
		return false
	}
	if actualRank != 0 {
		// Reference-array covariance needs reified descriptor planning beyond this
		// first bridge slice.
		return false
	}
	actualBase, _ := parseJavaTypeString(qualifyJavaTypeInDeclaringContext(actual, actualOwner))
	expectedBase, _ := parseJavaTypeString(qualifyJavaTypeInDeclaringContext(expected, expectedOwner))
	actualScope := resolveClassScopeByQualifiedName(classScopeCtx(actualOwner, ctx), actualBase)
	expectedScope := resolveClassScopeByQualifiedName(classScopeCtx(expectedOwner, ctx), expectedBase)
	return actualScope != nil && expectedScope != nil && javaReferenceTypeAssignable(actualScope, expectedScope, ctx)
}

// directOwnerOverrideBridgeRepresentationSupported is the whole-parameter
// commit gate. Every field, initializer, constructor, and callable family that
// carries a connected declaration must be representable before any one member
// is allowed to move to erased physical storage.
func directOwnerOverrideBridgeRepresentationSupported(
	rootPlan *directOwnerOverrideBridgeFamilyPlan,
	ctx Ctx,
) bool {
	if rootPlan == nil {
		return false
	}
	owner := rootPlan.owner
	method := rootPlan.method
	declarations := methodDirectOwnerTypeParameterDeclarations(owner, method)
	if len(declarations) == 0 {
		return false
	}
	related := make(map[*symbol.TypeParamDeclaration]struct{})
	for _, declaration := range declarations {
		for candidate := range callableTypeParameterDeclarationClosure(declaration, ctx) {
			related[candidate] = struct{}{}
		}
	}

	supported := true
	visitAllClassScopes(func(scope *symbol.ClassScope) bool {
		for declaration := range related {
			if !classScopeCarriesTypeParameterDeclaration(scope, declaration) {
				continue
			}
			if !ordinaryConcreteCallableOwner(scope) ||
				!classInitializerTypeSyntaxSupportsCallableErasure(scope, declaration, ctx) {
				supported = false
				return true
			}
			for _, field := range scope.Fields {
				if definitionTreeReferencesTypeParameter(field, declaration) &&
					!definitionTreeUsesOnlyBareTypeParameter(field, declaration, scope, false, ctx) {
					supported = false
					return true
				}
			}
			for _, candidateMethod := range scope.Methods {
				if candidateMethod == nil || candidateMethod.IsStatic {
					continue
				}
				if candidateMethod.Constructor {
					if !constructorBodySupportsCallableErasure(scope, candidateMethod, declaration, ctx) {
						supported = false
						return true
					}
					continue
				}
				if !methodBodyTypeSyntaxSupportsCallableErasure(scope, candidateMethod, declaration, true, ctx) {
					supported = false
					return true
				}
				if !methodDirectlyUsesTypeParameterDeclaration(candidateMethod, declaration) {
					continue
				}
				if !definitionTreeUsesOnlyBareTypeParameter(candidateMethod, declaration, scope, false, ctx) {
					supported = false
					return true
				}
				if directOwnerCallableMethodFamilyEligible(scope, candidateMethod, ctx) {
					continue
				}
				if directOwnerOverrideBridgePlanCoversMethod(rootPlan, scope, candidateMethod) {
					continue
				}
				if _, bridgeable := planDirectOwnerCallableOverrideBridgeFamilyUnchecked(scope, candidateMethod, ctx); !bridgeable {
					supported = false
					return true
				}
			}
		}
		return false
	})
	return supported
}

func directOwnerOverrideBridgePlanCoversMethod(
	plan *directOwnerOverrideBridgeFamilyPlan,
	owner *symbol.ClassScope,
	method *symbol.Definition,
) bool {
	if plan == nil || owner == nil || method == nil {
		return false
	}
	for _, bridge := range plan.overrides {
		if bridge.owner == owner && bridge.method == method {
			return true
		}
	}
	return false
}

func directOwnerOverrideBridgeExactExecutionName(plan directOwnerOverrideBridgePlan) string {
	if plan.method == nil {
		return ""
	}
	return collisionSafeExecutionIdentifier(plan.method.Name+"Java2goExactExecution", plan.owner)
}

func directOwnerOverrideBridgeErasedExecutionName(plan *directOwnerOverrideBridgeFamilyPlan) string {
	if plan == nil {
		return ""
	}
	return executionImplementationName(plan.method, plan.owner)
}
