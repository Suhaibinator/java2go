package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
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

// directOwnerTypeParameterFieldStorageType applies Java's erased heap layout
// to one bare class-owned type-parameter field. The generated struct is shared
// by every parameterized and raw view of the same Java object, so its physical
// slot cannot use Go's invariant T without rejecting legal raw heap pollution.
func directOwnerTypeParameterFieldStorageType(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	declared ast.Expr,
) ast.Expr {
	if definition == nil || definition.IsStatic {
		return declared
	}
	if _, ok := directOwnerTypeParameterForDefinition(owner, definition); !ok {
		return declared
	}
	return &ast.Ident{Name: "any"}
}

func sameExpressionSourceSite(left, right *sitter.Node) bool {
	return left != nil && right != nil && left.Type() == right.Type() &&
		left.StartByte() == right.StartByte() && left.EndByte() == right.EndByte()
}

func erasedFieldStorageTarget(ctx Ctx, node *sitter.Node) bool {
	return sameExpressionSourceSite(ctx.erasedFieldStorageTarget, node)
}

// directOwnerTypeParameterReadView replaces a type-variable result with its
// own first-bound erasure unless an exact target context requires the variable
// itself (for example `T previous = value` or `return value` from a T method).
// This mirrors Java bytecode: bound-only operations do not eagerly checkcast
// to the caller's concrete instantiation, while an exact T ABI boundary does.
func directOwnerTypeParameterReadView(javaType string, node *sitter.Node, ctx Ctx) string {
	if expectedTypeTargetsExpression(ctx, node) {
		if expected := strings.TrimSpace(ctx.expectedType); expected != "" && !isVarKeywordType(expected) {
			return readableWildcardProjection(expected)
		}
	}

	base, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	if rank != 0 {
		return javaType
	}
	base, arguments := parseJavaTypeString(base)
	if len(arguments) != 0 {
		return javaType
	}
	if parameter, ok := newTypeParameterLookup(visibleTypeParameterDeclarations(ctx)).byName[strings.TrimSpace(base)]; ok {
		return rawTypeParameterErasure(parameter, visibleTypeParameterDeclarations(ctx))
	}
	return readableWildcardProjection(javaType)
}

// directOwnerTypeParameterFieldRead exposes one erased field slot through the
// narrowest Go view required at this source read. The descriptor remains the
// field declaration's first-bound erasure even when the result view is a
// concrete parameterized type; ObjectView performs the nominal Java check and
// then resolves the appropriate generated subobject/interface view.
func directOwnerTypeParameterFieldRead(
	storage ast.Expr,
	node *sitter.Node,
	resolution *fieldResolution,
	readJavaType string,
	ctx Ctx,
) ast.Expr {
	if storage == nil || resolution == nil || resolution.def == nil || resolution.def.IsStatic ||
		erasedFieldStorageTarget(ctx, node) {
		return storage
	}
	use, ok := directOwnerTypeParameterForDefinition(resolution.owner, resolution.def)
	if !ok {
		return storage
	}

	resultJavaType := directOwnerTypeParameterReadView(readJavaType, node, ctx)
	if strings.TrimSpace(resultJavaType) == "" {
		resultJavaType = use.erasure
	}
	resultType := javaTypeStringToGoTypeExpr(resultJavaType, inScopeTypeParameters(ctx), ctx)
	resultType = abstractClassToInterface(resultType, resultJavaType, ctx)

	erasure := qualifyJavaTypeInDeclaringContext(use.erasure, resolution.owner)
	descriptor, descriptorOK := javaTypeDescriptorExpr(erasure, ctx)
	if !descriptorOK {
		return storage
	}
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{resultType}, []ast.Expr{storage, descriptor})
}
