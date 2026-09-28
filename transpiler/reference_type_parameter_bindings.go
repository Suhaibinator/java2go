package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Keep declaration contexts and hidden outer binders, unlike a source-name map
// that necessarily replaces an outer declaration when a method shadows it.
// Inferred declaration types use unique emitted names; bound syntax retains
// TypeParameterBindings and must be interpreted in its declaring context.
type referenceTypeParameterBinding struct {
	parameter symbol.TypeParam
	context   Ctx
}

func referenceTypeParameterBindings(ctx Ctx) []referenceTypeParameterBinding {
	var bindings []referenceTypeParameterBinding
	seen := map[typeParameterIdentityKey]bool{}
	appendParameters := func(parameters []symbol.TypeParam, declaring Ctx) {
		for _, parameter := range parameters {
			key := identityKeyForTypeParameter(parameter)
			if seen[key] {
				continue
			}
			seen[key] = true
			bindings = append(bindings, referenceTypeParameterBinding{parameter: parameter, context: declaring})
		}
	}
	if ctx.localScope == nil || !ctx.localScope.IsStatic {
		var owners []*symbol.ClassScope
		for scope := ctx.currentClass; scope != nil; scope = scope.Enclosing {
			owners = append(owners, scope)
		}
		for index := len(owners) - 1; index >= 0; index-- {
			owner := owners[index]
			appendParameters(owner.TypeParameters, classScopeCtx(owner, ctx))
		}
	}
	appendParameters(ctx.syntheticTypeParameters, ctx)
	if ctx.localScope != nil {
		appendParameters(ctx.localScope.TypeParameters, ctx)
	}
	return bindings
}

func resolveReferenceTypeParameter(javaType symbol.JavaType, ctx Ctx) (referenceTypeParameterBinding, bool) {
	base, rank := javaArrayTypeParts(strings.TrimSpace(javaType.Original))
	base, args := parseJavaTypeString(base)
	if rank != 0 || len(args) != 0 || strings.Contains(base, ".") {
		return referenceTypeParameterBinding{}, false
	}
	bindings := referenceTypeParameterBindings(ctx)
	if declaration := javaType.TypeParameterBindings[base]; declaration != nil {
		for _, binding := range bindings {
			if binding.parameter.Declaration == declaration {
				return binding, true
			}
		}
		return referenceTypeParameterBinding{}, false
	}
	// Emitted names on inferred declaration references are already bound. Keep
	// an outer emitted name from being stolen by an inner source-name alias.
	for index := len(bindings) - 1; index >= 0; index-- {
		if bindings[index].parameter.EmittedName() == base {
			return bindings[index], true
		}
	}
	for index := len(bindings) - 1; index >= 0; index-- {
		if bindings[index].parameter.Name == base {
			return bindings[index], true
		}
	}
	return referenceTypeParameterBinding{}, false
}

func invocationReferenceBindersCompatible(actual, expected string, ctx Ctx) bool {
	left, leftBound := resolveReferenceTypeParameter(symbol.JavaType{Original: actual}, ctx)
	right, rightBound := resolveReferenceTypeParameter(symbol.JavaType{Original: expected}, ctx)
	if !leftBound && !rightBound {
		return true
	}
	return leftBound && rightBound && identityKeyForTypeParameter(left.parameter) == identityKeyForTypeParameter(right.parameter)
}

// qualifyDeclaredReferenceType retains the declaration site of concrete names
// while leaving captured type-parameter references attached to their binders.
// Argument inference still runs in the caller; only the formal type crosses
// this boundary in canonical form.
func qualifyDeclaredReferenceType(javaType symbol.JavaType, ctx Ctx) string {
	var qualify func(string) string
	qualify = func(original string) string {
		base, rank := javaArrayTypeParts(strings.TrimSpace(original))
		suffix := strings.Repeat("[]", rank)
		for _, prefix := range []string{"? extends ", "? super "} {
			if strings.HasPrefix(base, prefix) {
				return prefix + qualify(strings.TrimPrefix(base, prefix)) + suffix
			}
		}
		base, args := parseJavaTypeString(base)
		var qualified string
		if declaration := javaType.TypeParameterBindings[base]; declaration != nil {
			qualified = declaration.GoName
			if qualified == "" {
				qualified = declaration.SourceName
			}
		} else if javaType.TypeParameterBindings == nil {
			if binding, found := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); found {
				qualified = binding.parameter.EmittedName()
			} else {
				qualified = qualifyDeclaredNominalReference(base, ctx)
			}
		} else {
			qualified = qualifyDeclaredNominalReference(base, ctx)
		}
		if len(args) > 0 {
			for index := range args {
				args[index] = qualify(args[index])
			}
			qualified += "<" + strings.Join(args, ", ") + ">"
		}
		return qualified + suffix
	}
	return qualify(javaType.Original)
}

func qualifyDeclaredNominalReference(base string, ctx Ctx) string {
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil {
		if qualified := qualifiedSourceClassName(scope); qualified != "" {
			return qualified
		}
	}
	if qualified, known := canonicalIntrinsicOwner(base, ctx); known {
		return qualified
	}
	if qualified, known := builtinThrowableReferenceName(base, ctx); known {
		return qualified
	}
	return base
}

func methodParameterReferenceType(parameter, method *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) string {
	declaring := classScopeCtx(owner, ctx)
	declaring.localScope = method
	declaring.syntheticTypeParameters = nil
	bindings := parameter.TypeParameterBindings
	if bindings == nil && parameter.DirectTypeParameter != nil {
		bindings = map[string]*symbol.TypeParamDeclaration{
			parameter.DirectTypeParameter.SourceName: parameter.DirectTypeParameter,
		}
	}
	return qualifyDeclaredReferenceType(symbol.JavaType{Original: parameter.OriginalType, TypeParameterBindings: bindings}, declaring)
}

func qualifyTypeParameterBounds(parameters []symbol.TypeParam, ctx Ctx) []symbol.TypeParam {
	result := append([]symbol.TypeParam(nil), parameters...)
	bindings := referenceTypeParameterBindings(ctx)
	for index, parameter := range result {
		declaring := ctx
		for _, binding := range bindings {
			if identityKeyForTypeParameter(binding.parameter) == identityKeyForTypeParameter(parameter) {
				declaring = binding.context
				break
			}
		}
		result[index].Bounds = append([]symbol.JavaType(nil), parameter.Bounds...)
		for boundIndex, bound := range parameter.Bounds {
			result[index].Bounds[boundIndex].Original = qualifyDeclaredReferenceType(bound, declaring)
		}
	}
	return result
}

// Applicability consumes both source syntax and declaration-qualified formals.
// Include the stable emitted aliases used for shadowed binders so loose
// invocation conversion can still recognize boxing into a candidate parameter.
func methodCandidateTypeParameterNames(owner *symbol.ClassScope, method *symbol.Definition) []string {
	var names []string
	appendParameters := func(parameters []symbol.TypeParam) {
		for _, parameter := range parameters {
			for _, name := range []string{parameter.Name, parameter.EmittedName()} {
				if name != "" && !containsString(names, name) {
					names = append(names, name)
				}
			}
		}
	}
	if owner != nil {
		appendParameters(owner.TypeParameters)
	}
	if method != nil {
		appendParameters(method.TypeParameters)
	}
	return names
}
