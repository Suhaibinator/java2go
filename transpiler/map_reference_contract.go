package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"strings"
)

// Java declaration ownership, rather than a Go method set, certifies map views.
func canonicalMapReferenceOwner(javaType string, ctx Ctx) string {
	base, rank := javaArrayTypeParts(javaType)
	if rank != 0 {
		return ""
	}
	base, _ = parseJavaTypeString(base)
	if !strings.Contains(base, ".") {
		if _, bound := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); bound {
			return ""
		}
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	if !known {
		return ""
	}
	switch owner {
	case "java.util.Map", "java.util.AbstractMap", "java.util.HashMap", "java.util.LinkedHashMap", "java.util.TreeMap", "java.util.concurrent.ConcurrentMap", "java.util.concurrent.ConcurrentHashMap":
		return owner
	}
	return ""
}
func mapReferenceArgumentIndependent(javaType string, ctx Ctx) bool {
	_, rank := javaArrayTypeParts(javaType)
	if rank != 0 {
		return false
	}
	_, args := parseJavaTypeString(javaType)
	owner := canonicalMapReferenceOwner(javaType, ctx)
	return (owner == "java.util.Map" || owner == "java.util.AbstractMap" || owner == "java.util.concurrent.ConcurrentMap") && (len(args) == 0 || len(args) == 2)
}
func mapIntrinsicLogicalName(name string) string {
	switch name {
	case "JavaMap":
		return "Map"
	case "JavaConcurrentMap":
		return "ConcurrentMap"
	case "JavaConcurrentHashMap":
		return "ConcurrentHashMap"
	}
	return name
}
func mapOwnerReaches(actual, expected string) bool {
	if actual == expected {
		return true
	}
	if expected == "java.util.Map" {
		return actual != ""
	}
	if expected == "java.util.HashMap" {
		return actual == "java.util.LinkedHashMap"
	}
	if expected == "java.util.AbstractMap" {
		return actual == "java.util.HashMap" || actual == "java.util.LinkedHashMap" || actual == "java.util.TreeMap" || actual == "java.util.concurrent.ConcurrentHashMap"
	}
	return expected == "java.util.concurrent.ConcurrentMap" && actual == "java.util.concurrent.ConcurrentHashMap"
}
func mapReferenceTypeArguments(typ symbol.JavaType, expected string, ctx Ctx, scopes map[*symbol.ClassScope]bool, binders map[typeParameterIdentityKey]bool) ([]string, bool) {
	base, rank := javaArrayTypeParts(typ.Original)
	if rank != 0 {
		return nil, false
	}
	if binding, found := resolveReferenceTypeParameter(typ, ctx); found {
		id := identityKeyForTypeParameter(binding.parameter)
		if binders[id] {
			return nil, false
		}
		binders[id] = true
		defer delete(binders, id)
		for _, bound := range binding.parameter.Bounds {
			qualifiedBound := symbol.JavaType{Original: qualifyDeclaredReferenceType(bound, binding.context)}
			if args, ok := mapReferenceTypeArguments(qualifiedBound, expected, binding.context, scopes, binders); ok {
				return args, true
			}
		}
		return nil, false
	}
	base, args := parseJavaTypeString(base)
	if owner := canonicalMapReferenceOwner(base, ctx); owner != "" {
		return args, mapOwnerReaches(owner, expected) && (len(args) == 0 || len(args) == 2)
	}
	owner := resolveClassScopeByQualifiedName(ctx, base)
	return mapReferenceScopeArguments(owner, args, expected, ctx, scopes, binders)
}
func mapReferenceScopeArguments(owner *symbol.ClassScope, args []string, expected string, ctx Ctx, scopes map[*symbol.ClassScope]bool, binders map[typeParameterIdentityKey]bool) ([]string, bool) {
	if owner == nil || scopes[owner] {
		return nil, false
	}
	bindings, ok := mapReferenceDeclarationBindings(owner, args, ctx)
	if !ok {
		return nil, false
	}
	scopes[owner] = true
	defer delete(scopes, owner)
	declaring := classHeaderTypeCtx(owner, ctx)
	for _, edge := range append(append([]string(nil), owner.ImplementedInterfaces...), owner.Superclass) {
		declared := symbol.JavaType{Original: edge, TypeParameterBindings: symbol.VisibleTypeParamBindings(owner.TypeParameters)}
		edge = substituteJavaTypeParams(qualifyDeclaredReferenceType(declared, declaring), bindings)
		if values, ok := mapReferenceTypeArguments(symbol.JavaType{Original: edge}, expected, declaring, scopes, binders); ok {
			if len(args) == 0 && len(owner.OwnTypeParameters()) != 0 {
				return nil, true
			}
			return values, true
		}
	}
	return nil, false
}

// Source arguments instantiate only a class's own declarations. Carried owner
// and executable parameters belong to the physical ABI and must be recovered
// by declaration identity, not by a repeated Java spelling such as T.
func mapReferenceDeclarationBindings(owner *symbol.ClassScope, args []string, ctx Ctx) (map[string]string, bool) {
	if owner == nil || len(args) > len(owner.TypeParameters) {
		return nil, false
	}
	parameters, own := owner.TypeParameters, owner.OwnTypeParameters()
	if len(own) > len(parameters) || (len(args) != 0 && len(args) != len(own) && len(args) != len(parameters)) {
		return nil, false
	}
	carried := len(parameters) - len(own)
	declarations := map[*symbol.TypeParamDeclaration]bool{}
	emitted := map[string]bool{}
	sourceNames := map[string]int{}
	for _, parameter := range parameters {
		sourceNames[parameter.Name]++
	}
	for index, parameter := range parameters {
		if parameter.EmittedName() == "" || emitted[parameter.EmittedName()] ||
			(parameter.Declaration == nil && (carried != 0 || sourceNames[parameter.Name] != 1)) {
			return nil, false
		}
		emitted[parameter.EmittedName()] = true
		if parameter.Declaration != nil {
			if declarations[parameter.Declaration] {
				return nil, false
			}
			declarations[parameter.Declaration] = true
		}
		if index >= carried {
			declared := own[index-carried]
			if parameter.Declaration != declared.Declaration || parameter.EmittedName() != declared.EmittedName() {
				return nil, false
			}
		}
	}
	actual := map[*symbol.TypeParamDeclaration]string{}
	if carried != 0 && len(args) != len(parameters) {
		for _, binding := range referenceTypeParameterBindings(ctx) {
			if binding.parameter.Declaration != nil {
				actual[binding.parameter.Declaration] = binding.parameter.EmittedName()
			}
		}
	}
	bindings := map[string]string{}
	for index, parameter := range parameters {
		var argument string
		switch {
		case len(args) == len(parameters):
			argument = args[index]
		case index < carried:
			argument = actual[parameter.Declaration]
			if argument == "" {
				return nil, false
			}
		case len(args) == len(own):
			argument = args[index-carried]
		default:
			argument = rawTypeParameterErasure(parameter, parameters)
			if len(parameter.Bounds) == 0 {
				argument = "java.lang.Object"
			}
		}
		if strings.TrimSpace(argument) == "" {
			return nil, false
		}
		bindings[parameter.EmittedName()] = argument
	}
	return bindings, true
}
func mapReferenceArguments(javaType, expected string, ctx Ctx) ([]string, bool) {
	return mapReferenceTypeArguments(symbol.JavaType{Original: qualifyDeclaredReferenceType(symbol.JavaType{Original: javaType}, ctx)}, expected, ctx, map[*symbol.ClassScope]bool{}, map[typeParameterIdentityKey]bool{})
}
func sourceMapReferenceAssignable(actual, expected string, params []string, ctx Ctx) bool {
	owner := canonicalMapReferenceOwner(expected, ctx)
	if owner != "java.util.Map" && owner != "java.util.AbstractMap" && owner != "java.util.concurrent.ConcurrentMap" {
		return false
	}
	args, ok := mapReferenceArguments(actual, owner, ctx)
	if !ok {
		return false
	}
	_, want := parseJavaTypeString(expected)
	return javaGenericArgumentsApplicable(args, want, params)
}
func sourceMapReferenceArguments(scope *symbol.ClassScope, expected string, ctx Ctx) ([]string, bool) {
	if scope == nil || scope.Class == nil {
		return nil, false
	}
	return mapReferenceScopeArguments(scope, scope.GoTypeParameterNames(), expected, ctx, map[*symbol.ClassScope]bool{}, map[typeParameterIdentityKey]bool{})
}
func sourceMapReferenceInterfaceIDs(scope *symbol.ClassScope, ctx Ctx) []ast.Expr {
	if scope == nil {
		return nil
	}
	declaring := classHeaderTypeCtx(scope, ctx)
	var ids []ast.Expr
	for _, edge := range scope.ImplementedInterfaces {
		switch canonicalMapReferenceOwner(edge, declaring) {
		case "java.util.Map":
			ids = append(ids, stdjavaQualifiedExpr("MapTypeID", ctx))
		case "java.util.concurrent.ConcurrentMap":
			ids = append(ids, stdjavaQualifiedExpr("ConcurrentMapTypeID", ctx))
		}
	}
	return ids
}
func mapReferenceProjection(value ast.Expr, target string, ctx Ctx) ast.Expr {
	owner := canonicalMapReferenceOwner(target, ctx)
	switch owner {
	case "java.util.Map", "java.util.AbstractMap":
		return stdjavaCall(ctx, "MapReferenceView", value)
	case "java.util.concurrent.ConcurrentMap":
		return stdjavaCall(ctx, "ConcurrentMapReferenceView", value)
	}
	return nil
}
