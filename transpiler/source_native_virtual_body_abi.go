package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Bounded ArrayList body carrier. This does not add a native family to general eligibility.
type sourceNativeVirtualBodyABI struct {
	owner            *symbol.ClassScope
	method           *symbol.Definition
	file             *symbol.FileScope
	declarations     map[*symbol.TypeParamDeclaration]bool
	erasedParameters []string
	erasedResult     string
}

func sourceNativeVirtualBindings(owner *symbol.ClassScope) map[string]*symbol.TypeParamDeclaration {
	bindings := symbol.VisibleTypeParamBindings(owner.TypeParameters)
	for _, parameter := range owner.TypeParameters {
		if parameter.Declaration != nil && bindings[parameter.EmittedName()] == nil {
			bindings[parameter.EmittedName()] = parameter.Declaration
		}
	}
	return bindings
}

func sourceNativeVirtualScalarDefinition(definition *symbol.Definition, declarations map[*symbol.TypeParamDeclaration]bool) bool {
	if definition == nil {
		return false
	}
	declared := definitionDeclaredJavaType(definition)
	base, rank := javaArrayTypeParts(strings.TrimSpace(declared.Original))
	name, args := parseJavaTypeString(base)
	return rank == 0 && len(args) == 0 && declarations[declared.TypeParameterBindings[name]] && definition.DirectTypeParameter == declared.TypeParameterBindings[name]
}

func (plan *sourceNativeVirtualBodyABI) scalar(javaType string, ctx Ctx) bool {
	if plan == nil || ctx.sourceNativeVirtualBody != plan || ctx.localScope != plan.method || ctx.currentClass != plan.owner || ctx.currentFile != plan.file {
		return false
	}
	if _, owned := supplierSourceDeclarationCtx(plan.owner, ctx); !owned {
		return false
	}
	base, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	name, args := parseJavaTypeString(base)
	if rank != 0 || len(args) != 0 || strings.ContainsAny(name, ".<?>") {
		return false
	}
	declaration := visibleTypeParameterDeclarationForJavaType(name, ctx)
	return declaration != nil && plan.declarations[declaration]
}

func sourceNativeVirtualBodyForActualDefinition(definition *symbol.Definition, ctx Ctx) *sourceNativeVirtualBodyABI {
	if definition == nil || definition.DeclarationNode == nil || definition.RuntimeDefault || definition.IsStatic {
		return nil
	}
	var owner *symbol.ClassScope
	for _, scope := range allSourceClassScopes() {
		for _, method := range scope.Methods {
			if method == definition {
				if owner != nil {
					return nil
				}
				owner = scope
			}
		}
	}
	return sourceNativeVirtualBodyForMethod(owner, definition, ctx)
}

func sourceNativeVirtualBodyForMethod(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) *sourceNativeVirtualBodyABI {
	if plan := sourceAbstractListVirtualBody(owner, method, ctx); plan != nil {
		return plan
	}
	return sourceArrayListScalarBody(owner, method, ctx)
}
