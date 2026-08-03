package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// directOwnerTypeParameterUse describes a rank-zero, bare use of a type
// parameter declared by the owning class. Java erases these uses at the class
// ABI boundary (fields, constructor parameters, and instance method
// parameters/results), while the parameterized source view determines where a
// caller must insert a narrowing cast.
//
// Declaration identity is deliberately authoritative. A method may shadow a
// class parameter with the same source name, and treating that method-owned T
// as the class-owned T would apply erasure at the wrong ABI boundary.
type directOwnerTypeParameterUse struct {
	parameter symbol.TypeParam
	erasure   string
}

// directOwnerTypeParameterForDefinition plans erasure for a definition whose
// complete Java type is one bare class type parameter. Arrays and nested uses
// such as T[] and List<T> have different runtime representations and are kept
// out of this first, intentionally narrow lowering block.
func directOwnerTypeParameterForDefinition(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
) (directOwnerTypeParameterUse, bool) {
	if owner == nil || definition == nil || definition.DirectTypeParameter == nil {
		return directOwnerTypeParameterUse{}, false
	}

	component, rank := javaArrayTypeParts(definition.OriginalType)
	if rank != 0 {
		return directOwnerTypeParameterUse{}, false
	}
	base, arguments := parseJavaTypeString(component)
	base = strings.TrimSpace(base)
	if base == "" || len(arguments) != 0 || strings.Contains(base, ".") {
		return directOwnerTypeParameterUse{}, false
	}

	for _, parameter := range owner.TypeParameters {
		// Identity, rather than spelling, distinguishes an owner parameter from a
		// same-named method/local parameter. Legacy parameters without declaration
		// identity cannot provide that proof and are conservatively excluded.
		if parameter.Declaration == nil || parameter.Declaration != definition.DirectTypeParameter {
			continue
		}
		if base != parameter.Name && base != parameter.EmittedName() {
			return directOwnerTypeParameterUse{}, false
		}
		return directOwnerTypeParameterUse{
			parameter: parameter,
			erasure:   rawTypeParameterErasure(parameter, owner.TypeParameters),
		}, true
	}

	return directOwnerTypeParameterUse{}, false
}

// directOwnerTypeParameterErasure is the string-level convenience form used
// by ABI and storage planners that do not need the declaration itself.
func directOwnerTypeParameterErasure(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
) (string, bool) {
	use, ok := directOwnerTypeParameterForDefinition(owner, definition)
	if !ok {
		return "", false
	}
	return use.erasure, true
}
