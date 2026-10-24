package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// isBuiltinCharSequence recognizes the declaration, not a matching short name.
// A source class/interface, explicit foreign import, or type parameter retains
// its own identity even when spelled CharSequence.
func isBuiltinCharSequence(javaType string, ctx Ctx) bool {
	base, arguments := parseJavaTypeString(strings.TrimSpace(javaType))
	if len(arguments) != 0 || (base != "CharSequence" && base != "java.lang.CharSequence") {
		return false
	}
	if base == "CharSequence" {
		if _, bound := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); bound {
			return false
		}
		if ctx.currentFile != nil {
			if owner, imported := ctx.currentFile.Imports[base]; imported && owner != "java.lang" {
				return false
			}
		}
	}
	return resolveClassScopeByQualifiedName(ctx, base) == nil
}

func sourceDirectCharSequence(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil {
		return false
	}
	declaring := classScopeCtx(scope, ctx)
	if isBuiltinCharSequence(scope.Superclass, declaring) {
		return true
	}
	for _, name := range scope.ImplementedInterfaces {
		if isBuiltinCharSequence(name, declaring) {
			return true
		}
	}
	return false
}

// Membership is transitive, but registration only adds direct edges. Each
// declaration resolves parents in its own file/lexical context, including a
// source subinterface extending the canonical builtin across package boundaries.
func sourceImplementsCharSequence(scope *symbol.ClassScope, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]bool{}
	var visit func(*symbol.ClassScope) bool
	visit = func(current *symbol.ClassScope) bool {
		if current == nil || seen[current] {
			return false
		}
		seen[current] = true
		if sourceDirectCharSequence(current, ctx) {
			return true
		}
		declaring := classScopeCtx(current, ctx)
		if visit(resolveSuperclassScopeInDeclaringContext(declaring, current)) {
			return true
		}
		for _, parent := range resolveImplementedInterfaceScopesInDeclaringContext(declaring, current) {
			if visit(parent) {
				return true
			}
		}
		return false
	}
	return visit(scope)
}

func sourceCharSequenceAssignable(actual, expected string, ctx Ctx) bool {
	if !isBuiltinCharSequence(expected, ctx) {
		return false
	}
	return charSequenceBoundAssignable(symbol.JavaType{Original: actual}, ctx, map[typeParameterIdentityKey]bool{})
}

// A bound is resolved at its declaration, not under names visible at a later
// call site. Captured identities also retain outer parameters hidden by method
// binders. Expected type parameters remain non-nominal in isBuiltinCharSequence.
func charSequenceBoundAssignable(actual symbol.JavaType, ctx Ctx, visiting map[typeParameterIdentityKey]bool) bool {
	base, rank := javaArrayTypeParts(strings.TrimSpace(actual.Original))
	if rank != 0 {
		return false
	}
	if binding, found := resolveReferenceTypeParameter(actual, ctx); found {
		identity := identityKeyForTypeParameter(binding.parameter)
		if visiting[identity] {
			return false
		}
		visiting[identity] = true
		defer delete(visiting, identity)
		for _, bound := range binding.parameter.Bounds {
			if charSequenceBoundAssignable(bound, binding.context, visiting) {
				return true
			}
		}
		return false
	}
	base, _ = parseJavaTypeString(base)
	// An unresolved captured dependency must never be rebound by its spelling.
	if actual.TypeParameterBindings[base] != nil {
		return false
	}
	if isBuiltinCharSequence(base, ctx) {
		return true
	}
	return sourceImplementsCharSequence(resolveClassScopeByQualifiedName(ctx, base), ctx)
}
