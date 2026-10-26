package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"strings"
)

var reflectProtocolConstants = map[string]string{
	"Type": "ReflectTypeTypeID", "ParameterizedType": "ParameterizedTypeTypeID",
	"GenericArrayType": "GenericArrayTypeTypeID", "WildcardType": "WildcardTypeTypeID",
	"TypeVariable": "TypeVariableTypeID", "GenericDeclaration": "GenericDeclarationTypeID",
	"AnnotatedElement": "AnnotatedElementTypeID",
}

func builtinReflectProtocol(javaType string, ctx Ctx) string {
	base, _ := parseJavaTypeString(javaType)
	if resolveClassScopeByQualifiedName(ctx, base) != nil {
		return ""
	}
	name := stripJavaQualifier(base)
	if _, ok := reflectProtocolConstants[name]; !ok {
		return ""
	}
	if strings.Contains(base, ".") && base != "java.lang.reflect."+name {
		return ""
	}
	return name
}
func isBuiltinReflectType(javaType string, ctx Ctx) bool {
	return builtinReflectProtocol(javaType, ctx) != ""
}
func reflectProtocolAssignable(actual, expected string) bool {
	if actual == expected {
		return true
	}
	switch expected {
	case "Type":
		return actual == "Class" || actual == "ParameterizedType" || actual == "GenericArrayType" || actual == "WildcardType" || actual == "TypeVariable"
	case "GenericDeclaration":
		return actual == "Class"
	case "AnnotatedElement":
		return actual == "Class" || actual == "TypeVariable" || actual == "GenericDeclaration"
	}
	return false
}
func sourceDirectReflectProtocols(scope *symbol.ClassScope, ctx Ctx) []string {
	if scope == nil {
		return nil
	}
	declaring := classScopeCtx(scope, ctx)
	names := append([]string{}, scope.ImplementedInterfaces...)
	names = append(names, scope.Superclass)
	var protocols []string
	for _, name := range names {
		if protocol := builtinReflectProtocol(name, declaring); protocol != "" {
			protocols = append(protocols, protocol)
		}
	}
	return protocols
}
func sourceImplementsReflectProtocol(scope *symbol.ClassScope, expected string, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]bool{}
	var implements func(*symbol.ClassScope) bool
	implements = func(current *symbol.ClassScope) bool {
		if current == nil || seen[current] {
			return false
		}
		seen[current] = true
		for _, actual := range sourceDirectReflectProtocols(current, ctx) {
			if expected == "" || reflectProtocolAssignable(actual, expected) {
				return true
			}
		}
		if implements(resolveSuperclassScopeInDeclaringContext(ctx, current)) {
			return true
		}
		for _, parent := range resolveImplementedInterfaceScopesInDeclaringContext(ctx, current) {
			if implements(parent) {
				return true
			}
		}
		return false
	}
	return implements(scope)
}
func sourceImplementsReflectType(scope *symbol.ClassScope, ctx Ctx) bool {
	return sourceImplementsReflectProtocol(scope, "", ctx)
}
func builtinReflectTypeAssignable(actual, expected string, ctx Ctx) bool {
	protocol := builtinReflectProtocol(expected, ctx)
	if protocol == "" {
		return false
	}
	base, _ := parseJavaTypeString(actual)
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil {
		return sourceImplementsReflectProtocol(scope, protocol, ctx)
	}
	actualProtocol := builtinReflectProtocol(actual, ctx)
	if base == "Class" || base == "java.lang.Class" {
		actualProtocol = "Class"
	}
	return reflectProtocolAssignable(actualProtocol, protocol)
}
