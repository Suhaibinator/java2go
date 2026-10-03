package transpiler

import (
	"go/ast"
	"go/types"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Java has separate type and value namespaces; Go locals can hide both types
// and the predeclared operations used by lowering. Keep Java lookup metadata
// intact and change only the emitted binding spelling.
func resolveLocalIdentifierHygiene() {
	reserved := localIdentifierReservedGlobals()
	for _, owner := range allSourceClassScopes() {
		for _, method := range owner.Methods {
			ctx := classScopeCtx(owner, Ctx{})
			ctx.localScope = method
			hygienizeLocalScopeWithReserved(ctx, reserved)
		}
	}
}

func hygienizeLocalScope(ctx Ctx) {
	hygienizeLocalScopeWithReserved(ctx, localIdentifierReservedGlobals())
}

func hygienizeLocalScopeWithReserved(ctx Ctx, reserved localIdentifierReservations) {
	if ctx.localScope == nil {
		return
	}
	for _, def := range append(append([]*symbol.Definition{}, ctx.localScope.Parameters...), ctx.localScope.Children...) {
		if def != nil {
			def.Name = hygienicLocalIdentifierWithReserved(def.Name, def.OriginalName, ctx, reserved)
		}
	}
}

// Build the whole-program reservation set once for a resolution pass; checking
// every method local must not repeatedly walk the entire dependency graph.
type localIdentifierReservations struct {
	globals      map[string]bool
	nominalTypes map[string]bool
}

func localIdentifierReservedGlobals() localIdentifierReservations {
	reserved := map[string]bool{}
	nominalTypes := map[string]bool{}
	for _, name := range types.Universe.Names() {
		reserved[name] = true
	}
	for name := range goKeywords {
		reserved[name] = true
	}
	// Fixed selectors emitted by standard-library lowering. Java/runtime imports
	// additionally allocate aliases against all source bindings in imports.go.
	for _, name := range strings.Fields("fmt math strconv strings regexp sort time os io sync atomic reflect utf8 utf16 bytes errors") {
		reserved[name] = true
	}
	for _, scope := range allSourceClassScopes() {
		if scope.Class != nil {
			nominalTypes[scope.Class.Name] = true
			nominalTypes[scope.Class.Name+"I"] = true
			nominalTypes[classDispatchTypeName(scope)] = true
			reserved["New"+scope.Class.Name] = true
		}
		for _, method := range scope.Methods {
			if method == nil {
				continue
			}
			if method.IsStatic || method.Constructor {
				reserved[method.Name] = true
			}
			if method.RequiresHelper {
				reserved[method.HelperName] = true
				reserved["New"+method.HelperName] = true
			}
		}
	}
	return localIdentifierReservations{globals: reserved, nominalTypes: nominalTypes}
}

func localIdentifierIsReserved(name string, ctx Ctx, reserved localIdentifierReservations) bool {
	// Constructor allocation, superclass arguments and generic projections can
	// introduce nominal types without spelling them in Java's body. Protect their
	// allocated Go names before any producer emits a body or a parameter binding.
	if reserved.nominalTypes[name] {
		return true
	}
	if reserved.globals[name] && localIdentifierRequiredByBody(name, ctx) {
		return true
	}
	for _, alias := range ctx.importAliases {
		if alias == name && localIdentifierRequiredByBody(name, ctx) {
			return true
		}
	}
	for _, parameter := range visibleTypeParameterDeclarations(ctx) {
		if parameter.EmittedName() == name && localIdentifierRequiredByBody(name, ctx) {
			return true
		}
	}
	return false
}

func hygienicLocalIdentifier(name, original string, ctx Ctx) string {
	return hygienicLocalIdentifierWithReserved(name, original, ctx, localIdentifierReservedGlobals())
}

func hygienicLocalIdentifierWithReserved(name, original string, ctx Ctx, reserved localIdentifierReservations) string {
	name = sanitizeGoIdent(name)
	if !localIdentifierIsReserved(name, ctx, reserved) {
		return name
	}
	stem := name + "Java2goLocal"
	for suffix := 0; ; suffix++ {
		candidate := stem
		if suffix > 0 {
			candidate += strconv.Itoa(suffix)
		}
		if localIdentifierIsReserved(candidate, ctx, reserved) {
			continue
		}
		if ctx.currentFile != nil && strings.Contains(string(ctx.currentFile.Source), candidate) {
			continue
		}
		if otherLocalHasGoName(ctx.localScope, original, candidate) {
			continue
		}
		return candidate
	}
}

func otherLocalHasGoName(def *symbol.Definition, original, candidate string) bool {
	if def == nil {
		return false
	}
	if def.OriginalName != original && sanitizeGoIdent(def.Name) == candidate {
		return true
	}
	for _, param := range def.Parameters {
		if otherLocalHasGoName(param, original, candidate) {
			return true
		}
	}
	for _, child := range def.Children {
		if otherLocalHasGoName(child, original, candidate) {
			return true
		}
	}
	return false
}

func localBindingName(name string, ctx Ctx) string {
	if ctx.localScope != nil {
		if def := ctx.localScope.FindVariable(name); def != nil {
			return sanitizeGoIdent(def.Name)
		}
	}
	return hygienicLocalIdentifier(name, name, ctx)
}

func localBindingIdent(node *sitter.Node, source []byte, ctx Ctx) *ast.Ident {
	return ast.NewIdent(localBindingName(node.Content(source), ctx))
}
