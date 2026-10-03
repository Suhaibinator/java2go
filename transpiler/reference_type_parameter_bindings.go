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
			appendParameters(owner.TypeParameters, classHeaderTypeCtx(owner, ctx))
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
	// Exact external names and single-type imports retain their declaration
	// identity even when that family has not migrated to intrinsic dispatch.
	if strings.Contains(base, ".") {
		return base
	}
	if ctx.currentFile != nil {
		if pkg, imported := ctx.currentFile.Imports[base]; imported {
			return pkg + "." + base
		}
	}
	if qualified, known := canonicalIntrinsicOwner(base, ctx); known {
		return qualified
	}
	if qualified, known := builtinThrowableReferenceName(base, ctx); known {
		return qualified
	}
	// These core reference declarations already have compiler/runtime type
	// models, but not all participate in the intrinsic-owner registry. Resolve
	// their implicit java.lang import only after source and explicit imports.
	switch base {
	case "Object", "String", "StringBuilder", "StringBuffer", "Class", "Enum",
		"Number", "Boolean", "Byte", "Short", "Character", "Integer", "Long", "Float", "Double",
		"Comparable", "CharSequence", "Cloneable", "Iterable", "AutoCloseable", "Appendable", "Readable",
		"Thread", "Runnable", "Deprecated":
		return "java.lang." + base
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
	qualified := qualifyDeclaredReferenceType(symbol.JavaType{Original: parameter.OriginalType, TypeParameterBindings: bindings}, declaring)
	return normalizeMethodMemberFormal(symbol.JavaType{Original: parameter.OriginalType, TypeParameterBindings: bindings}, qualified, owner, declaring)
}

// An implicit member type carries the enclosing declaration's parameters even
// though Java spells only its own arguments. Use declaration identities to
// recover those slots before substituting the invocation's receiver view.
func normalizeMethodMemberFormal(javaType symbol.JavaType, qualified string, owner *symbol.ClassScope, ctx Ctx) string {
	sourceBase, _ := javaArrayTypeParts(strings.TrimSpace(javaType.Original))
	base, rank := javaArrayTypeParts(strings.TrimSpace(qualified))
	suffix := strings.Repeat("[]", rank)
	for _, prefix := range []string{"? extends ", "? super "} {
		if strings.HasPrefix(sourceBase, prefix) {
			child := symbol.JavaType{Original: strings.TrimPrefix(sourceBase, prefix), TypeParameterBindings: javaType.TypeParameterBindings}
			return prefix + normalizeMethodMemberFormal(child, strings.TrimPrefix(base, prefix), owner, ctx) + suffix
		}
	}
	sourceName, sourceArguments := parseJavaTypeString(sourceBase)
	base, arguments := parseJavaTypeString(base)
	for index := range arguments {
		if index < len(sourceArguments) {
			child := symbol.JavaType{Original: sourceArguments[index], TypeParameterBindings: javaType.TypeParameterBindings}
			arguments[index] = normalizeMethodMemberFormal(child, arguments[index], owner, ctx)
		}
	}
	if javaType.TypeParameterBindings[sourceName] == nil {
		if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil && len(scope.TypeParameters) > len(scope.OwnTypeParameters()) {
			// A raw generic member formal permits unchecked invocation conversion;
			// its erasures are not invariant concrete source arguments.
			if len(arguments) > 0 || len(scope.OwnTypeParameters()) == 0 {
				if receiver, receiverArguments, typed := methodMemberFormalReceiver(javaType, sourceBase, scope, owner, ctx); typed {
					arguments = normalizeClassTypeArguments(scope, arguments, receiver, receiverArguments)
				}
			}
		}
	}
	if len(arguments) > 0 {
		base += "<" + strings.Join(arguments, ", ") + ">"
	}
	return base + suffix
}

// Resolve source qualifiers before their spelling is canonicalized. An explicit
// raw generic owner (Box.Member) differs from an implicit lexical Member even
// though both qualify to the same nominal name. Nongeneric intermediate members
// and fixed superclass views remain eligible for captured declaration slots.
func methodMemberFormalReceiver(javaType symbol.JavaType, original string, scope, owner *symbol.ClassScope, ctx Ctx) (*symbol.ClassScope, []string, bool) {
	receiver := owner
	var receiverArguments []string
	depth := 0
	for index := 0; index < len(original); index++ {
		switch original[index] {
		case '<':
			depth++
		case '>':
			depth--
		case '.':
			if depth != 0 {
				continue
			}
			prefix := original[:index]
			name, arguments := parseJavaTypeString(prefix)
			prefixScope := resolveClassScopeByQualifiedName(ctx, name)
			if prefixScope == nil {
				continue
			}
			if len(arguments) == 0 && len(prefixScope.OwnTypeParameters()) > 0 {
				return nil, nil, false
			}
			qualifiedPrefix := qualifyDeclaredReferenceType(symbol.JavaType{Original: prefix, TypeParameterBindings: javaType.TypeParameterBindings}, ctx)
			_, qualifiedArguments := parseJavaTypeString(qualifiedPrefix)
			carriedReceiver, carriedArguments, typed := methodMemberFormalCapturedReceiver(receiver, receiverArguments, prefixScope, len(qualifiedArguments), ctx)
			if !typed {
				return nil, nil, false
			}
			receiverArguments = normalizeClassTypeArguments(prefixScope, qualifiedArguments, carriedReceiver, carriedArguments)
			receiver = prefixScope
		}
	}
	_, supplied := parseJavaTypeString(original)
	return methodMemberFormalCapturedReceiver(receiver, receiverArguments, scope, len(supplied), ctx)
}

func methodMemberFormalCapturedReceiver(receiver *symbol.ClassScope, receiverArguments []string, scope *symbol.ClassScope, supplied int, ctx Ctx) (*symbol.ClassScope, []string, bool) {
	hidden := len(scope.TypeParameters) - len(scope.OwnTypeParameters())
	providedHidden := supplied - len(scope.OwnTypeParameters())
	complete := func(candidate *symbol.ClassScope, arguments []string) bool {
		available := receiverClassTypeArgumentBindings(candidate, arguments)
		for index := 0; index < hidden; index++ {
			if index < providedHidden {
				continue
			}
			if _, found := available.argumentFor(scope.TypeParameters[index]); !found {
				return false
			}
		}
		return true
	}
	if complete(receiver, receiverArguments) {
		return receiver, receiverArguments, true
	}
	if scope.Enclosing != nil {
		mapped := mapClassTypeArgumentStringsToAncestor(receiver, receiverArgumentsOrDeclaration(receiver, receiverArguments), scope.Enclosing, ctx)
		if mapped != nil && complete(scope.Enclosing, mapped) {
			return scope.Enclosing, mapped, true
		}
	}
	return nil, nil, false
}

func receiverArgumentsOrDeclaration(scope *symbol.ClassScope, arguments []string) []string {
	if arguments == nil && scope != nil {
		return scope.GoTypeParameterNames()
	}
	return arguments
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
