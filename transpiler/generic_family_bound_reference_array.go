package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

// A real raw source occurrence demands one storage layout for every alias of
// its complete family. Named declarations, binder identities and all slot/body
// audits remain prerequisites; an unsupported bound alone grants no layout.
func genericFamilySourceBoundRawDemand(plan *genericFamilyPlan, ctx Ctx) bool {
	for _, seed := range sourceGenericViewDemandSeedsFromTree(ctx, true) {
		if _, member := plan.members[seed]; member {
			return true
		}
	}
	return false
}

// Reified reference arrays already have one physical runtime type. Admit only
// constructor formals tied to this exact class declaration, never arbitrary
// invariant fields or nested generic containers.
func genericFamilySourceBoundConstructorArray(owner *symbol.ClassScope, definition *symbol.Definition, declaration *symbol.TypeParamDeclaration) bool {
	if owner == nil || definition == nil || declaration == nil {
		return false
	}
	component, rank := javaArrayTypeParts(strings.TrimSpace(definition.OriginalType))
	base, args := parseJavaTypeString(component)
	if rank == 0 || len(args) != 0 || definition.TypeParameterBindings[base] != declaration {
		return false
	}
	for _, method := range owner.Methods {
		if method == nil || !method.Constructor || method.DeclarationNode == nil {
			continue
		}
		for _, formal := range method.Parameters {
			if formal == definition {
				return true
			}
		}
	}
	return false
}
func (plan *genericFamilyPlan) sourceBoundReferenceArray(declaration *symbol.TypeParamDeclaration, rank int) bool {
	representation, present := plan.representations[declaration]
	return rank > 0 && present && representation.sourceBound != nil
}

// The logical assignment conversion runs first. Transport that converted value
// to the field's certified first-bound storage without losing its shared Java
// identity, evaluating it twice, or dereferencing a null reference.
func genericFamilySourceBoundFieldStore(value ast.Expr, left *sitter.Node, logical string, ctx Ctx, source []byte) ast.Expr {
	if value == nil {
		return value
	}
	_, erasure, known := directOwnerFieldAccessView(left, ctx, source)
	if !known {
		return value
	}
	expected := resolveClassScopeByQualifiedName(ctx, erasure)
	actual := resolveClassScopeByQualifiedName(ctx, logical)
	if expected == nil || expected.IsInterface || actual == nil || actual == expected || !classInheritsFrom(actual, expected, ctx) {
		return value
	}
	return nullableSuperclassEdgeView(value, logical, erasure, actual, expected, ctx)
}
