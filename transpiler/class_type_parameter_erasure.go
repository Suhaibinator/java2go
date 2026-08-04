package transpiler

import (
	"go/ast"
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

func directOwnerInterfaceErasure(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	ctx Ctx,
) (string, bool) {
	use, ok := directOwnerTypeParameterForDefinition(owner, definition)
	if !ok || len(use.parameter.Bounds) != 1 {
		return "", false
	}
	erasure := qualifyJavaTypeInDeclaringContext(use.erasure, owner)
	base, _ := parseJavaTypeString(erasure)
	erasureScope := resolveClassScopeByQualifiedName(ctx, base)
	if erasureScope == nil || !erasureScope.IsInterface {
		return "", false
	}
	return erasure, true
}

// directOwnerStandaloneInterfaceErasure limits the first physical member ABI
// slice to standalone concrete leaf classes. Generic interfaces, abstract
// classes, implemented interfaces, superclass overrides, and known subclasses
// need Java-style bridge methods before erased field/result storage and source
// views can safely coexist in Go.
func directOwnerStandaloneInterfaceErasure(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	ctx Ctx,
) (string, bool) {
	if owner == nil || definition == nil || definition.IsStatic || owner.IsInterface ||
		owner.IsAbstract || owner.IsEnum || owner.IsInner || len(owner.ImplementedInterfaces) != 0 ||
		strings.TrimSpace(owner.Superclass) != "" || classHasKnownSubclass(owner, ctx) {
		return "", false
	}
	for _, nested := range owner.Subclasses {
		if nested != nil && nested.IsInner {
			return "", false
		}
	}
	return directOwnerInterfaceErasure(owner, definition, ctx)
}

func directOwnerOrdinaryFieldInterfaceErasure(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	ctx Ctx,
) (string, bool) {
	if owner == nil || owner.Class == nil || definition == nil || definition.DeclarationNode == nil {
		return "", false
	}
	classNode := owner.Class.DeclarationNode
	if classNode == nil {
		return "", false
	}
	parent := classNode.Parent()
	if classNode.Type() != "class_declaration" || parent == nil ||
		(parent.Type() != "program" && parent.Type() != "class_body") ||
		definition.DeclarationNode.Type() != "field_declaration" {
		return "", false
	}
	return directOwnerStandaloneInterfaceErasure(owner, definition, ctx)
}

func directOwnerOrdinaryMethodInterfaceErasure(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	ctx Ctx,
) (string, bool) {
	if owner == nil || owner.Class == nil || definition == nil || definition.DeclarationNode == nil {
		return "", false
	}
	classNode := owner.Class.DeclarationNode
	if classNode == nil {
		return "", false
	}
	parent := classNode.Parent()
	if classNode.Type() != "class_declaration" || parent == nil ||
		(parent.Type() != "program" && parent.Type() != "class_body") ||
		definition.DeclarationNode.Type() != "method_declaration" {
		return "", false
	}
	return directOwnerStandaloneInterfaceErasure(owner, definition, ctx)
}

// directOwnerTypeParameterFieldStorageType gives a bare class-owned type
// parameter field the Java declaration's erased representation. Every raw and
// parameterized alias consequently shares one physical slot type; use-site
// lowering is responsible for restoring the narrower source view where javac
// would emit a checkcast.
func directOwnerTypeParameterFieldStorageType(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	declared ast.Expr,
	ctx Ctx,
) ast.Expr {
	if definition == nil || definition.IsStatic {
		return declared
	}
	erasure, ok := directOwnerOrdinaryFieldInterfaceErasure(owner, definition, ctx)
	if !ok {
		return declared
	}
	// This first physical-storage slice is limited to generated interface
	// erasures. A Go interface can retain every implementing object's identity,
	// including heap pollution through a raw alias. Concrete-class erasures need
	// the broader canonical reference/subobject migration, while unbounded Object
	// fields retain the established generic Go API until that migration lands.
	storage := javaTypeStringToGoTypeExpr(erasure, inScopeTypeParameters(ctx), ctx)
	return abstractClassToInterface(storage, erasure, ctx)
}

// directOwnerTypeParameterMethodResultType applies Java's erased callable ABI
// to a method whose complete result is one bare parameter declared by its
// owning class. The source-level parameterized view remains in symbol metadata;
// callers insert a concrete ObjectView only where javac would emit checkcast.
func directOwnerTypeParameterMethodResultType(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	declared ast.Expr,
	ctx Ctx,
) ast.Expr {
	if definition == nil || definition.IsStatic {
		return declared
	}
	erasure, ok := directOwnerOrdinaryMethodInterfaceErasure(owner, definition, ctx)
	if !ok {
		return declared
	}
	result := javaTypeStringToGoTypeExpr(erasure, inScopeTypeParameters(ctx), ctx)
	return abstractClassToInterface(result, erasure, ctx)
}
