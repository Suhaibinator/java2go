package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// Native reflected members and generated annotation proxies own their complete
// protocols. Until a separate source implementor bridge is proved, reject a
// user extension instead of embedding an uninitialized native interface.
func reflectionSourceCarrierUnsupported(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil || scope.Class == nil {
		return false
	}
	if node := scope.Class.DeclarationNode; node != nil && node.Type() == "annotation_type_declaration" {
		return false
	}

	// Each source definition is checked independently by strict conversion.
	// An indirect extension is refused at its unsupported supplied root; avoid
	// a second hierarchy walk for every unrelated class or interface.
	if scope.Superclass == "" && len(scope.ImplementedInterfaces) == 0 {
		return false
	}
	declaring := classScopeCtx(scope, ctx)
	check := func(written string) bool {
		base, _ := parseJavaTypeString(written)
		if base == "" {
			return false
		}
		if owner, owned := canonicalIntrinsicOwner(base, declaring); owned && (owner == "java.lang.reflect.AccessibleObject" || owner == "java.lang.reflect.Member" || owner == "java.lang.annotation.Annotation") {
			return true
		}
		source := resolveClassScopeByQualifiedName(declaring, base)
		return source != nil && source.Class != nil && source.Class.DeclarationNode != nil && source.Class.DeclarationNode.Type() == "annotation_type_declaration"
	}
	if check(scope.Superclass) {
		return true
	}
	for _, implemented := range scope.ImplementedInterfaces {
		if check(implemented) {
			return true
		}
	}
	return false
}
func reflectionAnonymousCarrierUnsupported(javaType string, ctx Ctx) bool {
	base, _ := parseJavaTypeString(javaType)
	if owner, owned := canonicalIntrinsicOwner(base, ctx); owned && (owner == "java.lang.reflect.AccessibleObject" || owner == "java.lang.reflect.Member" || owner == "java.lang.annotation.Annotation") {
		return true
	}
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil {
		if scope.Class != nil && scope.Class.DeclarationNode != nil && scope.Class.DeclarationNode.Type() == "annotation_type_declaration" {
			return true
		}
		return reflectionSourceCarrierUnsupported(scope, ctx)
	}
	return false
}
