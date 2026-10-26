package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Enum's recursive Java argument constrains source calls, not its erased
// physical object. Only the canonical declaration receives the runtime view.
func isBuiltinEnum(javaType string, ctx Ctx) bool {
	base, args := parseJavaTypeString(strings.TrimSpace(javaType))
	if len(args) > 1 || (base != "Enum" && base != "java.lang.Enum") {
		return false
	}
	if base == "Enum" {
		if _, found := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); found {
			return false
		}
		if ctx.currentFile != nil {
			if owner, found := ctx.currentFile.Imports[base]; found && owner != "java.lang" {
				return false
			}
		}
	}
	return resolveClassScopeByQualifiedName(ctx, base) == nil
}

func enumReferenceType(javaType string, ctx Ctx) bool {
	return enumBoundReference(symbol.JavaType{Original: javaType}, ctx, map[typeParameterIdentityKey]bool{})
}
func enumBoundReference(typ symbol.JavaType, ctx Ctx, visiting map[typeParameterIdentityKey]bool) bool {
	base, rank := javaArrayTypeParts(typ.Original)
	if rank != 0 {
		return false
	}
	if binding, found := resolveReferenceTypeParameter(typ, ctx); found {
		key := identityKeyForTypeParameter(binding.parameter)
		if visiting[key] {
			return false
		}
		visiting[key] = true
		defer delete(visiting, key)
		for _, bound := range binding.parameter.Bounds {
			if enumBoundReference(bound, binding.context, visiting) {
				return true
			}
		}
		return false
	}
	base, _ = parseJavaTypeString(base)
	if typ.TypeParameterBindings[base] != nil {
		return false
	}
	if isBuiltinEnum(typ.Original, ctx) {
		return true
	}
	scope := resolveClassScopeByQualifiedName(ctx, base)
	return scope != nil && scope.IsEnum
}

func enumReferenceAssignable(actual, expected string, ctx Ctx) bool {
	if !enumReferenceType(actual, ctx) {
		return false
	}
	base, _ := parseJavaTypeString(expected)
	if overrideBridgeCanonicalObjectResult(expected, base, ctx) {
		return true
	}
	if isBuiltinEnum(expected, ctx) {
		_, want := parseJavaTypeString(expected)
		if len(want) == 0 || want[0] == "?" {
			return true
		}
		base, have := parseJavaTypeString(actual)
		if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil && scope.IsEnum {
			have = []string{javaClassBinaryName(scope)}
		}
		if len(have) != 1 {
			return false
		}
		return javaGenericArgumentsApplicable(have, want, nil)
	}
	return false
}
