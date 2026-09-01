package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
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

type directOwnerSpecializedOverrideBridgeSelection struct {
	family *directOwnerOverrideBridgeFamilyPlan
	bridge directOwnerOverrideBridgePlan
}

type directOwnerOverrideBridgeMethodVisibility uint8

const (
	directOwnerOverrideBridgePackagePrivate directOwnerOverrideBridgeMethodVisibility = iota
	directOwnerOverrideBridgeProtected
	directOwnerOverrideBridgePublic
)

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
	if directOwnerOverrideBridgeHasIncompatibleDispatchFamily(plan, ctx) {
		return nil, false
	}
	if !directOwnerOverrideBridgeRepresentationSupported(plan, ctx) {
		return nil, false
	}
	return plan, true
}

// directOwnerOverrideBridgeHasIncompatibleDispatchFamily rejects a method that
// simultaneously participates in bridge families which cannot share one Go
// dispatch entry. Different JVM erasures are incompatible. So are identical
// erasures whose generated selectors have different Go identities, as when a
// package-private Java override widens to public and changes mJava2goExecution
// into MJava2goExecution. Emitting either family alone would silently leave the
// other dispatch path on a partially migrated ABI.
//
// Use unchecked plans while auditing the connected family. Calling the public
// planner here would recurse through this same all-or-nothing gate.
func directOwnerOverrideBridgeHasIncompatibleDispatchFamily(
	rootPlan *directOwnerOverrideBridgeFamilyPlan,
	ctx Ctx,
) bool {
	if rootPlan == nil || rootPlan.owner == nil || rootPlan.method == nil {
		return true
	}
	type familyMember struct {
		owner  *symbol.ClassScope
		method *symbol.Definition
	}
	members := []familyMember{{owner: rootPlan.owner, method: rootPlan.method}}
	membersValid := true
	visitAllClassScopes(func(descendant *symbol.ClassScope) bool {
		if descendant == nil || descendant == rootPlan.owner ||
			!classScopeDescendsFrom(descendant, rootPlan.owner, ctx) {
			return false
		}
		ancestorArgs := mapClassTypeArgumentStringsToAncestor(
			descendant,
			descendant.GoTypeParameterNames(),
			rootPlan.owner,
			ctx,
		)
		if len(ancestorArgs) != len(rootPlan.owner.TypeParameters) {
			membersValid = false
			return true
		}
		mappedParameters, mappedResult := mappedOverrideSourceSignature(
			rootPlan.owner,
			rootPlan.method,
			ancestorArgs,
		)
		matching := directDeclaredOverrides(
			rootPlan.owner,
			descendant,
			rootPlan.method,
			mappedParameters,
			mappedResult,
			ctx,
		)
		if len(matching) > 1 {
			membersValid = false
			return true
		}
		if len(matching) == 1 {
			// Retain declarations whose erased descriptor is unchanged. They need no
			// javac bridge and therefore are absent from rootPlan.overrides, but an
			// access-widening rename can still split the Go selector family.
			members = append(members, familyMember{owner: descendant, method: matching[0]})
		}
		return false
	})
	if !membersValid {
		return true
	}

	for _, member := range members {
		if member.owner == nil || member.method == nil {
			return true
		}

		// A specialized declaration can itself root a second family for its
		// descendants. That is safe only when both families share one erased
		// descriptor, as in a generic intermediate with an unchanged bound.
		if member.owner != rootPlan.owner || member.method != rootPlan.method {
			if nested, ok := planDirectOwnerCallableOverrideBridgeFamilyUnchecked(member.owner, member.method, ctx); ok &&
				nested != nil && nested.requiresErasedView &&
				!directOwnerOverrideBridgeDispatchFamiliesIdentical(rootPlan, nested, ctx) {
				return true
			}
		}

		// The root itself may already be a specialized bridge declaration for
		// an ancestor with a wider erasure. Find only ancestor plans that
		// actually cover this exact declaration; same-name overloads do not
		// connect the families.
		seen := make(map[*symbol.ClassScope]struct{})
		memberCtx := classScopeCtx(member.owner, ctx)
		for ancestor := resolveSuperclassScopeInDeclaringContext(memberCtx, member.owner); ancestor != nil; ancestor = resolveSuperclassScopeInDeclaringContext(memberCtx, ancestor) {
			if _, duplicate := seen[ancestor]; duplicate {
				return true
			}
			seen[ancestor] = struct{}{}
			for _, candidate := range ancestor.Methods {
				if candidate == nil || candidate.Constructor || candidate.IsStatic || candidate.IsPrivate ||
					candidate.OriginalName != member.method.OriginalName ||
					len(candidate.Parameters) != len(member.method.Parameters) {
					continue
				}
				ancestorPlan, ok := planDirectOwnerCallableOverrideBridgeFamilyUnchecked(ancestor, candidate, ctx)
				if !ok || ancestorPlan == nil || !ancestorPlan.requiresErasedView ||
					!directOwnerOverrideBridgeFamilyDeclaresOverride(
						ancestorPlan,
						member.owner,
						member.method,
						ctx,
					) {
					continue
				}
				if !directOwnerOverrideBridgeDispatchFamiliesIdentical(rootPlan, ancestorPlan, ctx) {
					return true
				}
			}
		}
	}
	return false
}

func directOwnerOverrideBridgeFamilyDeclaresOverride(
	plan *directOwnerOverrideBridgeFamilyPlan,
	overrideOwner *symbol.ClassScope,
	overrideMethod *symbol.Definition,
	ctx Ctx,
) bool {
	if plan == nil || plan.owner == nil || plan.method == nil || overrideOwner == nil || overrideMethod == nil {
		return false
	}
	if plan.owner == overrideOwner && plan.method == overrideMethod {
		return true
	}
	return directOwnerOverrideBridgeMethodDeclaresOverride(
		plan.owner,
		plan.method,
		overrideOwner,
		overrideMethod,
		ctx,
	)
}

func directOwnerOverrideBridgeMethodDeclaresOverride(
	ancestorOwner *symbol.ClassScope,
	ancestorMethod *symbol.Definition,
	overrideOwner *symbol.ClassScope,
	overrideMethod *symbol.Definition,
	ctx Ctx,
) bool {
	if ancestorOwner == nil || ancestorMethod == nil || overrideOwner == nil || overrideMethod == nil ||
		!classScopeDescendsFrom(overrideOwner, ancestorOwner, ctx) {
		return false
	}
	ancestorArgs := mapClassTypeArgumentStringsToAncestor(
		overrideOwner,
		overrideOwner.GoTypeParameterNames(),
		ancestorOwner,
		ctx,
	)
	if len(ancestorArgs) != len(ancestorOwner.TypeParameters) {
		return false
	}
	mappedParameters, mappedResult := mappedOverrideSourceSignature(ancestorOwner, ancestorMethod, ancestorArgs)
	matching := directDeclaredOverrides(
		ancestorOwner,
		overrideOwner,
		ancestorMethod,
		mappedParameters,
		mappedResult,
		ctx,
	)
	return len(matching) == 1 && matching[0] == overrideMethod
}

func directOwnerOverrideBridgeDispatchFamiliesIdentical(
	left *directOwnerOverrideBridgeFamilyPlan,
	right *directOwnerOverrideBridgeFamilyPlan,
	ctx Ctx,
) bool {
	return directOwnerOverrideBridgeDescriptorsIdentical(left, right, ctx) &&
		directOwnerOverrideBridgeSelectorIdentity(left) == directOwnerOverrideBridgeSelectorIdentity(right)
}

// directOwnerOverrideBridgeSelectorIdentity models Go method identity. Exported
// selectors are shared across packages; unexported selectors include the
// declaring package in their identity. Use the ordinary family root name here,
// not executionImplementationName, because that public helper consults bridge
// selection and would recurse into this all-or-nothing compatibility gate.
func directOwnerOverrideBridgeSelectorIdentity(plan *directOwnerOverrideBridgeFamilyPlan) string {
	if plan == nil || plan.owner == nil || plan.method == nil {
		return ""
	}
	return directOwnerOverrideBridgeMethodSelectorIdentity(plan.owner, plan.method)
}

func directOwnerOverrideBridgeMethodSelectorIdentity(owner *symbol.ClassScope, method *symbol.Definition) string {
	if owner == nil || method == nil {
		return ""
	}
	name := collisionSafeExecutionIdentifier(method.Name+executionMethodSuffix, owner)
	if ast.IsExported(name) {
		return "exported\x00" + name
	}
	return "package\x00" + findJavaPackageForClassScope(owner) + "\x00" + name
}

func directOwnerOverrideBridgeDescriptorsIdentical(
	left *directOwnerOverrideBridgeFamilyPlan,
	right *directOwnerOverrideBridgeFamilyPlan,
	ctx Ctx,
) bool {
	if left == nil || right == nil || left.owner == nil || right.owner == nil ||
		len(left.erasedParameters) != len(right.erasedParameters) {
		return false
	}
	for index := range left.erasedParameters {
		if !overrideBridgeJavaTypesIdentical(
			left.erasedParameters[index],
			left.owner,
			right.erasedParameters[index],
			right.owner,
			ctx,
		) {
			return false
		}
	}
	return overrideBridgeJavaTypesIdentical(
		left.erasedResult,
		left.owner,
		right.erasedResult,
		right.owner,
		ctx,
	)
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
		matching := directDeclaredOverrides(owner, descendant, method, mappedParameters, mappedResult, ctx)
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
		if directOwnerOverrideBridgeVisibility(owner, method) == directOwnerOverrideBridgeProtected &&
			!directOwnerOverrideBridgeSamePackage(owner, descendant) {
			// Java permits a protected override across packages, but its current
			// generated hidden selector is unexported and therefore has a different
			// Go package identity. Reject the entire physical family until protected
			// execution selectors have an exported, collision-safe representation.
			valid = false
			return true
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
	ancestorOwner *symbol.ClassScope,
	owner *symbol.ClassScope,
	ancestorMethod *symbol.Definition,
	mappedParameters []string,
	mappedResult string,
	ctx Ctx,
) []*symbol.Definition {
	if ancestorOwner == nil || owner == nil || ancestorMethod == nil {
		return nil
	}
	if directOwnerOverrideBridgeVisibility(ancestorOwner, ancestorMethod) == directOwnerOverrideBridgePackagePrivate &&
		!directOwnerOverrideBridgeSamePackage(ancestorOwner, owner) {
		// A same-signature declaration in a different Java package does not
		// override or inherit a package-private ancestor method. The raw Base entry
		// remains valid and dispatches to Base; do not synthesize a Child bridge.
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

func directOwnerOverrideBridgeVisibility(
	owner *symbol.ClassScope,
	method *symbol.Definition,
) directOwnerOverrideBridgeMethodVisibility {
	if owner != nil && owner.IsInterface {
		return directOwnerOverrideBridgePublic
	}
	if directOwnerOverrideBridgeMethodHasModifier(method, "public") {
		return directOwnerOverrideBridgePublic
	}
	if directOwnerOverrideBridgeMethodHasModifier(method, "protected") {
		return directOwnerOverrideBridgeProtected
	}
	return directOwnerOverrideBridgePackagePrivate
}

func directOwnerOverrideBridgeMethodHasModifier(method *symbol.Definition, want string) bool {
	if method == nil || method.DeclarationNode == nil || want == "" {
		return false
	}
	declaration := method.DeclarationNode
	for index := 0; index < int(declaration.ChildCount()); index++ {
		modifiers := declaration.Child(index)
		if modifiers == nil || modifiers.Type() != "modifiers" {
			continue
		}
		for modifierIndex := 0; modifierIndex < int(modifiers.ChildCount()); modifierIndex++ {
			modifier := modifiers.Child(modifierIndex)
			if modifier != nil && modifier.Type() == want {
				return true
			}
		}
	}
	return false
}

func directOwnerOverrideBridgeSamePackage(left, right *symbol.ClassScope) bool {
	if left == nil || right == nil {
		return false
	}
	return findJavaPackageForClassScope(left) == findJavaPackageForClassScope(right)
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
	if resultNeedsWidening && !overrideBridgePlainResultWideningSupported(erasedResult, ancestorOwner, ctx) {
		// Go pointers do not support Java's concrete-class covariance. Keep that
		// wider bridge family gated until it has a representation-only superclass
		// projection; ObjectView would add a nominal runtime check javac's areturn
		// does not perform. Interface/Object erasures are plain Go assignments.
		return directOwnerOverrideBridgePlan{}, false, false
	}
	plan.result = directOwnerOverrideBridgeResultPlan{
		erasedJavaType:   erasedResult,
		sourceJavaType:   qualifyJavaTypeInDeclaringContext(mappedResult, overrideOwner),
		overrideJavaType: overrideResult,
		requiresWidening: resultNeedsWidening,
	}
	bridgeRequired = bridgeRequired || resultDescriptorDiffers
	return plan, bridgeRequired, true
}

func overrideBridgePlainResultWideningSupported(javaType string, owner *symbol.ClassScope, ctx Ctx) bool {
	component, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	if rank != 0 {
		return false
	}
	base, _ := parseJavaTypeString(component)
	if stripJavaQualifier(base) == "Object" {
		return true
	}
	scope := resolveClassScopeByQualifiedName(classScopeCtx(owner, ctx), base)
	return scope != nil && scope.IsInterface
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

// directOwnerSpecializedOverrideBridgeForMethod finds the erased ancestor
// descriptor implemented by one narrow source override. Walking from the
// immediate superclass keeps the exact body associated with its nearest Java
// declaration. A deeper generic chain can expose the same erased descriptor
// more than once; identical plans are deduplicated, while incompatible bridge
// descriptors remain conservatively unsupported because Go cannot overload
// one stable selector.
func directOwnerSpecializedOverrideBridgeForMethod(
	owner *symbol.ClassScope,
	method *symbol.Definition,
	ctx Ctx,
) (directOwnerSpecializedOverrideBridgeSelection, bool) {
	if owner == nil || method == nil || method.Constructor || method.IsStatic || method.IsPrivate {
		return directOwnerSpecializedOverrideBridgeSelection{}, false
	}
	ctx = classScopeCtx(owner, ctx)
	var selected directOwnerSpecializedOverrideBridgeSelection
	descriptorKey := ""
	seen := make(map[*symbol.ClassScope]struct{})
	for ancestor := resolveSuperclassScopeInDeclaringContext(ctx, owner); ancestor != nil; ancestor = resolveSuperclassScopeInDeclaringContext(ctx, ancestor) {
		if _, duplicate := seen[ancestor]; duplicate {
			return directOwnerSpecializedOverrideBridgeSelection{}, false
		}
		seen[ancestor] = struct{}{}
		for _, candidate := range ancestor.Methods {
			if candidate == nil || candidate.Constructor || candidate.IsStatic || candidate.IsPrivate ||
				candidate.OriginalName != method.OriginalName || len(candidate.Parameters) != len(method.Parameters) {
				continue
			}
			family, ok := planDirectOwnerCallableOverrideBridgeFamily(ancestor, candidate, ctx)
			if !ok || family == nil {
				continue
			}
			for _, bridge := range family.overrides {
				if bridge.owner != owner || bridge.method != method {
					continue
				}
				key := overrideBridgeFamilyDescriptorKey(family, ctx)
				if descriptorKey == "" {
					descriptorKey = key
					selected = directOwnerSpecializedOverrideBridgeSelection{family: family, bridge: bridge}
					continue
				}
				if key != descriptorKey {
					return directOwnerSpecializedOverrideBridgeSelection{}, false
				}
			}
		}
	}
	return selected, selected.family != nil
}

func overrideBridgeFamilyDescriptorKey(plan *directOwnerOverrideBridgeFamilyPlan, ctx Ctx) string {
	if plan == nil || plan.owner == nil {
		return ""
	}
	declarationCtx := classScopeCtx(plan.owner, ctx)
	descriptor := make([]string, 0, len(plan.erasedParameters)+1)
	for _, parameter := range plan.erasedParameters {
		descriptor = append(descriptor, callablePhysicalTypeKey(parameter, declarationCtx))
	}
	descriptor = append(descriptor, callablePhysicalTypeKey(plan.erasedResult, declarationCtx))
	return strings.Join(descriptor, "\x00") + "\x01" + directOwnerOverrideBridgeSelectorIdentity(plan)
}

func directOwnerOverrideBridgeFamilyUsesErasedHiddenOnly(
	owner *symbol.ClassScope,
	method *symbol.Definition,
	ctx Ctx,
) bool {
	_, ok := planDirectOwnerCallableOverrideBridgeFamily(owner, method, ctx)
	return ok
}

// buildDirectOwnerOverrideBridgeMethodDecls replaces the ordinary paired
// public/hidden method lowering only for bridge-planned families. Generic
// ancestors keep their source-shaped Go wrapper while their hidden body uses
// the erased JVM descriptor. Specialized overrides keep their narrow wrapper
// and narrow hidden body, plus a stable erased bridge used by ancestor dispatch.
func buildDirectOwnerOverrideBridgeMethodDecls(
	declaration *ast.FuncDecl,
	sourceParams *ast.FieldList,
	sourceResults *ast.FieldList,
	executionName string,
	ctx Ctx,
) ([]ast.Decl, bool) {
	if declaration == nil || ctx.currentClass == nil || ctx.localScope == nil || ctx.localScope.IsStatic {
		return nil, false
	}
	if family, ok := planDirectOwnerCallableOverrideBridgeFamily(ctx.currentClass, ctx.localScope, ctx); ok {
		return buildDirectOwnerErasedFamilyMethodDecls(
			declaration,
			sourceParams,
			sourceResults,
			executionName,
			family,
			ctx,
		), true
	}
	if selection, ok := directOwnerSpecializedOverrideBridgeForMethod(ctx.currentClass, ctx.localScope, ctx); ok {
		exactName := directOwnerOverrideBridgeExactExecutionName(selection.bridge)
		declarations := buildExecutionAwareFuncDecls(declaration, exactName, executionName, ctx)
		if bridge := buildDirectOwnerSpecializedOverrideBridgeDecl(declaration, executionName, selection, ctx); bridge != nil {
			declarations = append(declarations, bridge)
		}
		return declarations, true
	}
	return nil, false
}

func buildDirectOwnerErasedFamilyMethodDecls(
	declaration *ast.FuncDecl,
	sourceParams *ast.FieldList,
	sourceResults *ast.FieldList,
	executionName string,
	family *directOwnerOverrideBridgeFamilyPlan,
	ctx Ctx,
) []ast.Decl {
	if declaration == nil || family == nil {
		return nil
	}
	implementationName := directOwnerOverrideBridgeErasedExecutionName(family)
	declarations := buildExecutionAwareFuncDecls(declaration, implementationName, executionName, ctx)
	if len(declarations) < 2 {
		return declarations
	}

	arguments := append([]ast.Expr{newExecutionExpr(ctx)}, methodCallArgs(sourceParams)...)
	call := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: declaration.Recv.List[0].Names[0].Name},
			Sel: &ast.Ident{Name: implementationName},
		},
		Args: arguments,
	}
	var statements []ast.Stmt
	if sourceResults != nil && len(sourceResults.List) > 0 {
		result := ast.Expr(call)
		if _, direct := directOwnerTypeParameterForDefinition(family.owner, family.method); direct {
			if descriptor, ok := javaTypeDescriptorExpr(family.erasedResult, ctx); ok {
				result = stdjavaGenericCall(
					ctx,
					"ObjectView",
					[]ast.Expr{sourceResults.List[0].Type},
					[]ast.Expr{result, descriptor},
				)
			}
		}
		statements = append(statements, &ast.ReturnStmt{Results: []ast.Expr{result}})
	} else {
		statements = append(statements, &ast.ExprStmt{X: call})
	}

	declarations[0] = &ast.FuncDecl{
		Doc:  declaration.Doc,
		Name: &ast.Ident{Name: declaration.Name.Name},
		Recv: cloneFieldList(declaration.Recv),
		Type: &ast.FuncType{
			Params:  cloneFieldList(sourceParams),
			Results: cloneFieldList(sourceResults),
		},
		Body: &ast.BlockStmt{List: statements},
	}
	return declarations
}

func buildDirectOwnerSpecializedOverrideBridgeDecl(
	declaration *ast.FuncDecl,
	executionName string,
	selection directOwnerSpecializedOverrideBridgeSelection,
	ctx Ctx,
) ast.Decl {
	if declaration == nil || declaration.Recv == nil || len(declaration.Recv.List) == 0 ||
		len(declaration.Recv.List[0].Names) == 0 || selection.family == nil || selection.bridge.method == nil {
		return nil
	}

	bridgeCtx := classScopeCtx(selection.bridge.owner, ctx)
	params := &ast.FieldList{List: []*ast.Field{executionParameterField(executionName, bridgeCtx)}}
	usedNames := map[string]struct{}{executionName: {}}
	for index, parameter := range selection.bridge.method.Parameters {
		name := parameter.Name
		usedNames[name] = struct{}{}
		javaType := selection.family.erasedParameters[index]
		parameterType := javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(bridgeCtx), bridgeCtx)
		parameterType = abstractClassToInterface(parameterType, javaType, bridgeCtx)
		params.List = append(params.List, &ast.Field{
			Names: []*ast.Ident{{Name: name}},
			Type:  parameterType,
		})
	}

	var results *ast.FieldList
	if selection.family.erasedResult != "void" {
		resultType := javaTypeStringToGoTypeExpr(selection.family.erasedResult, inScopeTypeParameters(bridgeCtx), bridgeCtx)
		resultType = abstractClassToInterface(resultType, selection.family.erasedResult, bridgeCtx)
		results = &ast.FieldList{List: []*ast.Field{{Type: resultType}}}
	}

	receiverName := declaration.Recv.List[0].Names[0].Name
	body := []ast.Stmt{instanceMethodNilReceiverGuard(receiverName)}
	arguments := make([]ast.Expr, len(selection.bridge.parameters))
	for index, parameter := range selection.bridge.parameters {
		argument := ast.Expr(&ast.Ident{Name: selection.bridge.method.Parameters[index].Name})
		if parameter.requiresCast {
			castName := synchronizedUniqueLocalName("__java2goBridgeArg"+strconv.Itoa(index), usedNames)
			targetType := javaTypeStringToGoTypeExpr(parameter.overrideJavaType, inScopeTypeParameters(bridgeCtx), bridgeCtx)
			descriptor, ok := javaTypeDescriptorExpr(parameter.overrideJavaType, bridgeCtx)
			if !ok {
				return nil
			}
			body = append(body, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: castName}},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{stdjavaGenericCall(
					bridgeCtx,
					"ObjectView",
					[]ast.Expr{targetType},
					[]ast.Expr{argument, descriptor},
				)},
			})
			argument = &ast.Ident{Name: castName}
		}
		arguments[index] = argument
	}

	call := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: receiverName},
			Sel: &ast.Ident{Name: directOwnerOverrideBridgeExactExecutionName(selection.bridge)},
		},
		Args: append([]ast.Expr{&ast.Ident{Name: executionName}}, arguments...),
	}
	if results != nil {
		// javac's synthetic bridge returns the exact body's reference with areturn;
		// it does not checkcast on an upcast. The planner admits only interface or
		// Object widening here, both of which are ordinary Go assignability.
		body = append(body, &ast.ReturnStmt{Results: []ast.Expr{call}})
	} else {
		body = append(body, &ast.ExprStmt{X: call})
	}

	return &ast.FuncDecl{
		Name: &ast.Ident{Name: directOwnerOverrideBridgeErasedExecutionName(selection.family)},
		Recv: cloneFieldList(declaration.Recv),
		Type: &ast.FuncType{Params: params, Results: results},
		Body: &ast.BlockStmt{List: body},
	}
}
