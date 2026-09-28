package transpiler

import "github.com/NickyBoy89/java2go/symbol"

func isBuiltinReflectType(javaType string, ctx Ctx) bool {
	base, args := parseJavaTypeString(javaType)
	return len(args) == 0 && (base == "Type" || base == "java.lang.reflect.Type") && resolveClassScopeByQualifiedName(ctx, base) == nil
}

func sourceDirectlyImplementsReflectType(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil {
		return false
	}
	declaring := classScopeCtx(scope, ctx)
	for _, name := range scope.ImplementedInterfaces {
		if isBuiltinReflectType(name, declaring) {
			return true
		}
	}
	return isBuiltinReflectType(scope.Superclass, declaring)
}

func sourceImplementsReflectType(scope *symbol.ClassScope, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]bool{}
	var implements func(*symbol.ClassScope) bool
	implements = func(current *symbol.ClassScope) bool {
		if current == nil || seen[current] {
			return false
		}
		seen[current] = true
		if sourceDirectlyImplementsReflectType(current, ctx) {
			return true
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

func builtinReflectTypeAssignable(actual, expected string, ctx Ctx) bool {
	if !isBuiltinReflectType(expected, ctx) {
		return false
	}
	base, _ := parseJavaTypeString(actual)
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil {
		return sourceImplementsReflectType(scope, ctx)
	}
	return base == "Class" || base == "java.lang.Class" || isBuiltinReflectType(actual, ctx)
}
