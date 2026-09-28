package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// leafObjectMemberErasureEligible plans Object-valued instance slots and method
// descriptors together. It does not equate different Go instantiations: raw
// aliases retain their original object pointer until canonical layout lowering
// is available. Source signatures and binder identities remain untouched.
func leafObjectMemberErasureEligible(owner *symbol.ClassScope, ctx Ctx) bool {
	if !ordinaryConcreteCallableOwner(owner) || owner.IsInner || len(owner.TypeParameters) == 0 ||
		len(owner.Subclasses) != 0 || strings.TrimSpace(owner.Superclass) != "" ||
		classHasKnownSubclass(owner, ctx) || classHasUnmodeledCallableSubclass(owner, ctx) {
		return false
	}
	for _, parameter := range owner.TypeParameters {
		if parameter.Declaration == nil || len(parameter.Bounds) > 1 ||
			stripJavaQualifier(rawTypeParameterErasure(parameter, owner.TypeParameters)) != "Object" {
			return false
		}
		declaration := parameter.Declaration
		if !classInitializerTypeSyntaxSupportsCallableErasure(owner, declaration, ctx) {
			return false
		}
		for _, field := range owner.Fields {
			if field == nil || field.IsStatic {
				continue
			}
			if !definitionTreeUsesOnlyBareTypeParameter(field, declaration, owner, false, ctx) {
				return false
			}
		}
		for _, method := range owner.Methods {
			if method == nil || method.IsStatic {
				continue
			}
			if len(method.TypeParameters) != 0 || !definitionTreeUsesOnlyBareTypeParameter(method, declaration, owner, false, ctx) {
				return false
			}
			if method.Constructor {
				if !constructorBodySupportsCallableErasure(owner, method, declaration, ctx) {
					return false
				}
				continue
			}
			uses := methodDirectlyUsesTypeParameterDeclaration(method, declaration)
			if uses && !ordinarySourceMethod(owner, method) {
				return false
			}
			if !methodBodyTypeSyntaxSupportsCallableErasure(owner, method, declaration, uses, ctx) {
				return false
			}
		}
	}
	return true
}

func leafObjectErasure(owner *symbol.ClassScope, erasure string, ctx Ctx) bool {
	return (erasure == "Object" || erasure == "java.lang.Object") &&
		resolveClassScopeByQualifiedName(classScopeCtx(owner, ctx), erasure) == nil &&
		leafObjectMemberErasureEligible(owner, ctx)
}
