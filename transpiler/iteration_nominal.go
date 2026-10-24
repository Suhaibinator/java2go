package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

func init() {
	registerIntrinsicOwner("java.lang.Iterable", true)
	registerIntrinsicOwner("java.util.Iterator", true)
}

// Runtime iteration protocols denote canonical declarations, never a source
// type or a visible type parameter with the same spelling.
func canonicalIterationOwner(javaType string, ctx Ctx) string {
	base, _ := parseJavaTypeString(javaType)
	if !strings.Contains(base, ".") {
		if _, found := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); found {
			return ""
		}
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	if known && (owner == "java.lang.Iterable" || owner == "java.util.Iterator") {
		return owner
	}
	return ""
}

func isExternalIterableType(javaType string, ctx Ctx) bool {
	return canonicalIterationOwner(javaType, ctx) == "java.lang.Iterable"
}

// iterationElementType retains Java substitution while the physical protocol
// erases its element. Parent edges are resolved in their declaration context.
func iterationElementType(javaType, target string, ctx Ctx) (string, bool) {
	return iterationElementTypeVisit(symbol.JavaType{Original: javaType}, target, ctx, map[*symbol.ClassScope]bool{}, map[typeParameterIdentityKey]bool{})
}

func iterationElementTypeVisit(typ symbol.JavaType, target string, ctx Ctx, scopes map[*symbol.ClassScope]bool, binders map[typeParameterIdentityKey]bool) (string, bool) {
	base, rank := javaArrayTypeParts(typ.Original)
	if rank != 0 {
		return "", false
	}
	if binding, found := resolveReferenceTypeParameter(typ, ctx); found {
		identity := identityKeyForTypeParameter(binding.parameter)
		if binders[identity] {
			return "", false
		}
		binders[identity] = true
		defer delete(binders, identity)
		for _, bound := range binding.parameter.Bounds {
			if element, ok := iterationElementTypeVisit(bound, target, binding.context, scopes, binders); ok {
				return element, true
			}
		}
		return "", false
	}
	base, args := parseJavaTypeString(base)
	if canonicalIterationOwner(base, ctx) == target {
		if len(args) == 1 {
			return args[0], true
		}
		if len(args) == 0 {
			return "java.lang.Object", true
		}
		return "", false
	}
	if element, ok := builtinCollectionIterationElementType(typ.Original, target, ctx); ok {
		return element, true
	}
	owner := resolveClassScopeByQualifiedName(ctx, base)
	if owner == nil || scopes[owner] {
		return "", false
	}
	scopes[owner] = true
	defer delete(scopes, owner)
	declaring := classScopeCtx(owner, ctx)
	bindings := map[string]string{}
	for index, parameter := range owner.TypeParameters {
		argument := rawTypeParameterErasure(parameter, owner.TypeParameters)
		if index < len(args) {
			argument = args[index]
		}
		bindings[parameter.Name] = argument
		bindings[parameter.EmittedName()] = argument
	}
	parents := append(append([]string(nil), owner.ImplementedInterfaces...), owner.Superclass)
	for _, parent := range parents {
		parent = substituteJavaTypeParams(qualifyJavaTypeInDeclaringContext(parent, owner), bindings)
		if element, ok := iterationElementTypeVisit(symbol.JavaType{Original: parent}, target, declaring, scopes, binders); ok {
			return element, true
		}
	}
	return "", false
}

func sourceIterationContract(scope *symbol.ClassScope, target string, ctx Ctx) (string, bool) {
	if scope == nil || scope.Class == nil {
		return "", false
	}
	declaring := classScopeCtx(scope, ctx)
	bindings := map[string]string{}
	for _, p := range scope.TypeParameters {
		bindings[p.Name] = p.EmittedName()
	}
	for _, parent := range append(append([]string(nil), scope.ImplementedInterfaces...), scope.Superclass) {
		parent = substituteJavaTypeParams(qualifyJavaTypeInDeclaringContext(parent, scope), bindings)
		if element, ok := iterationElementType(parent, target, declaring); ok {
			return element, true
		}
	}
	return "", false
}

func iterationProtocolReservedSelector(name string) bool {
	if abstractMapProtocolReservedSelector(name) {
		return true
	}
	if mapEntryProtocolReservedSelector(name) {
		return true
	}
	if abstractCollectionProtocolReservedSelector(name) {
		return true
	}
	switch name {
	case "IteratorJava2goExecution", "HasNextJava2goExecution", "NextJava2goExecution", "IteratorRemoveJava2goExecution":
		return true
	}
	return false
}
