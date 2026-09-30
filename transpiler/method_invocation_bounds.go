package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Argument-shape compatibility alone cannot make a bounded generic candidate
// applicable. Keep inferred argument types before Go's erasure view is chosen,
// and check the candidate's Java bounds in their declaration context.
type methodInferenceRelation uint8

const (
	methodInferenceLower methodInferenceRelation = iota
	methodInferenceEqual
	methodInferenceUpper
)

type methodInferenceConstraint struct {
	typ      string
	relation methodInferenceRelation
	capture  bool
}

func methodInvocationBoundsApplicable(method *symbol.Definition, owner *symbol.ClassScope, arguments []*sitter.Node, fixedArray bool, ctx Ctx, source []byte) bool {
	_, applicable := methodInvocationConstraintBindings(method, owner, arguments, fixedArray, ctx, source)
	return applicable
}

// Return bindings backed by actual argument constraints, leaving null-only or
// otherwise unconstrained variables to the invocation's existing completion.
func methodInvocationConstraintBindings(method *symbol.Definition, owner *symbol.ClassScope, arguments []*sitter.Node, fixedArray bool, ctx Ctx, source []byte) (map[string]string, bool) {
	if len(method.TypeParameters) == 0 {
		return nil, true
	}
	declaring := classScopeCtx(owner, ctx)
	declaring.localScope = method
	declaring.syntheticTypeParameters = nil
	parameters := qualifyTypeParameterBounds(method.TypeParameters, declaring)
	names := make(map[string]struct{}, len(parameters))
	for _, parameter := range parameters {
		names[parameter.EmittedName()] = struct{}{}
	}
	inference := ctx.Clone()
	inference.expectedType = ""
	inference.expectedTypeRoot = nil
	constraints := make(map[string][]methodInferenceConstraint)
	for index, argument := range arguments {
		parameterIndex := index
		last := len(method.Parameters) - 1
		if last >= 0 && executionParameterIsVariadic(method, last) && parameterIndex > last {
			parameterIndex = last
		}
		if parameterIndex >= len(method.Parameters) || method.Parameters[parameterIndex] == nil {
			continue
		}
		actual, known := inferExprJavaType(argument, inference, source)
		unwrapped := unwrapParenthesizedExpressionNode(argument)
		if !known || actual == "" || actual == ternaryNullJavaType || unwrapped != nil && unwrapped.Type() == "null_literal" {
			continue
		}
		actual = qualifyDeclaredReferenceType(symbol.JavaType{Original: actual}, inference)
		formal := methodParameterReferenceType(method.Parameters[parameterIndex], method, owner, ctx)
		if fixedArray && parameterIndex == last {
			formal += "[]"
		}
		collectMethodApplicabilityConstraints(formal, actual, methodInferenceLower, names, constraints, inference)
	}
	bindings := make(map[string]string, len(parameters))
	for _, parameter := range parameters {
		var lower []string
		var equality *methodInferenceConstraint
		for _, constraint := range constraints[parameter.EmittedName()] {
			switch constraint.relation {
			case methodInferenceEqual:
				if equality != nil && (equality.capture || constraint.capture ||
					!javaInferenceSameType(equality.typ, constraint.typ, inference) ||
					!invocationReferenceBindersCompatible(equality.typ, constraint.typ, inference)) {
					return nil, false
				}
				copy := constraint
				equality = &copy
			case methodInferenceLower:
				lower = append(lower, constraint.typ)
			}
		}
		inferred := javaInferenceLeastUpperBound(lower, inference)
		if equality != nil {
			inferred = equality.typ
			for _, actual := range lower {
				// A capture has no known lower bound; a concrete value cannot
				// be inserted merely because it meets the capture's upper bound.
				if equality.capture || !invocationActualSatisfiesBound(actual, inferred, inference, make(map[typeParameterIdentityKey]bool)) {
					return nil, false
				}
			}
		}
		if inferred == "" {
			inferred = rawTypeParameterErasure(parameter, parameters)
			// With only upper constraints, Object (or the declared erasure)
			// need not be a feasible instantiation. Select a more specific
			// supplied upper bound, then validate every upper and declaration
			// bound below. Do not combine invariant arguments using this rule.
			for _, constraint := range constraints[parameter.EmittedName()] {
				if constraint.relation == methodInferenceUpper &&
					!invocationActualSatisfiesBound(inferred, constraint.typ, inference, make(map[typeParameterIdentityKey]bool)) {
					inferred = constraint.typ
				}
			}
		}
		bindings[parameter.EmittedName()] = inferred
	}
	for _, parameter := range parameters {
		inferred := bindings[parameter.EmittedName()]
		for _, constraint := range constraints[parameter.EmittedName()] {
			if constraint.relation == methodInferenceUpper && !invocationActualSatisfiesBound(inferred, constraint.typ, inference, make(map[typeParameterIdentityKey]bool)) {
				return nil, false
			}
		}
		for _, bound := range parameter.Bounds {
			if !inferredMethodBoundSatisfied(inferred, bound.Original, parameters, bindings, inference, make(map[typeParameterIdentityKey]bool)) {
				return nil, false
			}
		}
	}
	observed := make(map[string]string)
	for _, parameter := range parameters {
		if len(constraints[parameter.EmittedName()]) != 0 {
			observed[parameter.Name] = bindings[parameter.EmittedName()]
		}
	}
	return observed, true
}

// Invariant type arguments impose equality, whereas a bare value parameter
// contributes a lower bound. Wildcards change the relation explicitly; they
// must not turn two incompatible invariant arguments into a common superclass.
func collectMethodApplicabilityConstraints(formal, actual string, relation methodInferenceRelation, names map[string]struct{}, constraints map[string][]methodInferenceConstraint, ctx Ctx) {
	formal, actual = strings.TrimSpace(formal), strings.TrimSpace(actual)
	if formal == "" || actual == "" || formal == "?" {
		return
	}
	if strings.HasPrefix(formal, "? extends ") {
		collectMethodApplicabilityConstraints(strings.TrimPrefix(formal, "? extends "), inferenceCaptureUpperBound(actual), methodInferenceLower, names, constraints, ctx)
		return
	}
	if strings.HasPrefix(formal, "? super ") {
		if actual == "?" || strings.HasPrefix(actual, "? extends ") {
			return
		}
		collectMethodApplicabilityConstraints(strings.TrimPrefix(formal, "? super "), strings.TrimPrefix(actual, "? super "), methodInferenceUpper, names, constraints, ctx)
		return
	}
	formalBase, formalRank := javaArrayTypeParts(formal)
	actualBase, actualRank := javaArrayTypeParts(actual)
	formalRaw, formalArguments := parseJavaTypeString(formalBase)
	if _, variable := names[formalRaw]; variable && len(formalArguments) == 0 {
		if actualRank < formalRank {
			return
		}
		typ := actualBase + strings.Repeat("[]", actualRank-formalRank)
		capture := strings.HasPrefix(typ, "?")
		typ = inferenceCaptureUpperBound(typ)
		if actualRank == 0 && formalRank == 0 {
			if boxed := ternaryBoxedJavaType(typ); boxed != "" {
				typ = "java.lang." + boxed
			}
		}
		constraints[formalRaw] = append(constraints[formalRaw], methodInferenceConstraint{typ: typ, relation: relation, capture: capture})
		return
	}
	if formalRank != actualRank {
		return
	}
	actualRaw, actualArguments := parseJavaTypeString(actualBase)
	if len(formalArguments) == 0 || len(formalArguments) != len(actualArguments) {
		return
	}
	formalScope := resolveClassScopeByQualifiedName(ctx, formalRaw)
	actualScope := resolveClassScopeByQualifiedName(ctx, actualRaw)
	if formalScope != nil || actualScope != nil {
		if formalScope == nil || formalScope != actualScope {
			return
		}
	} else if !sameJavaRawType(formalRaw, actualRaw) && !builtinJavaReferenceAssignable(actualRaw, formalRaw, ctx) {
		return
	}
	for index := range formalArguments {
		collectMethodApplicabilityConstraints(formalArguments[index], actualArguments[index], methodInferenceEqual, names, constraints, ctx)
	}
}

// The generated view of one capture is its readable upper bound. This does
// not make it interchangeable with another capture or allow writes of the
// bound type; applicability retains capture identity separately above.
func inferenceCaptureUpperBound(actual string) string {
	actual = strings.TrimSpace(actual)
	if strings.HasPrefix(actual, "? extends ") {
		return strings.TrimSpace(strings.TrimPrefix(actual, "? extends "))
	}
	if actual == "?" || strings.HasPrefix(actual, "? super ") {
		return "java.lang.Object"
	}
	return actual
}

// A variable that occurs only as another variable's upper bound still carries
// all of its declared bounds. Follow that declaration instead of replacing it
// with its first-bound Go erasure (which would discard intersection bounds).
func inferredMethodBoundSatisfied(actual, expected string, parameters []symbol.TypeParam, inferred map[string]string, ctx Ctx, visiting map[typeParameterIdentityKey]bool) bool {
	expected = substituteJavaTypeParameters(expected, inferred)
	for _, parameter := range parameters {
		if expected != parameter.EmittedName() {
			continue
		}
		key := identityKeyForTypeParameter(parameter)
		if visiting[key] {
			return false
		}
		visiting[key] = true
		defer delete(visiting, key)
		for _, bound := range parameter.Bounds {
			if !inferredMethodBoundSatisfied(actual, bound.Original, parameters, inferred, ctx, visiting) {
				return false
			}
		}
		return true
	}
	return invocationActualSatisfiesBound(actual, expected, ctx, make(map[typeParameterIdentityKey]bool))
}

func invocationActualSatisfiesBound(actual, expected string, ctx Ctx, visiting map[typeParameterIdentityKey]bool) bool {
	if javaInferenceTypeAssignable(actual, expected, ctx) {
		return true
	}
	binding, found := resolveReferenceTypeParameter(symbol.JavaType{Original: actual}, ctx)
	if !found {
		return false
	}
	key := identityKeyForTypeParameter(binding.parameter)
	if visiting[key] {
		return false
	}
	visiting[key] = true
	defer delete(visiting, key)
	for _, bound := range binding.parameter.Bounds {
		qualified := qualifyDeclaredReferenceType(bound, binding.context)
		if invocationActualSatisfiesBound(qualified, expected, binding.context, visiting) {
			return true
		}
	}
	return false
}

// Invocation inference reads values in the caller, but concrete formal names
// and bounds belong to the method declaration. Keep those contexts separate
// before choosing a Go erasure or mapping an inferred method argument.
func invocationMethodDeclarationContext(method *symbol.Definition, caller Ctx) (Ctx, *symbol.ClassScope) {
	owner := classScopeOwningMethodDefinition(method)
	if owner == nil {
		return caller, nil
	}
	declaring := classScopeCtx(owner, caller)
	declaring.localScope = method
	declaring.syntheticTypeParameters = nil
	return declaring, owner
}
