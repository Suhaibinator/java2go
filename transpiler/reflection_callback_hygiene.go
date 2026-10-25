package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// Static reflection callbacks have generated bodies, so their free names are
// the actual emitted storage/type expressions and import aliases. Reserve those
// Go identifiers, together with physical source members, before binding locals.
// Inspecting the generated Go AST avoids rescanning Java syntax per field and
// also catches renamed types/backing variables that differ from source spelling.
func sourceReflectionCallbackUsedNames(scope *symbol.ClassScope, ctx Ctx, free ...ast.Expr) map[string]struct{} {
	used := make(map[string]struct{})
	if scope != nil {
		if scope.Class != nil {
			used[scope.Class.Name] = struct{}{}
		}
		for _, field := range scope.Fields {
			if field != nil {
				used[field.Name] = struct{}{}
			}
		}
	}
	// Materialize the actual runtime qualifier before collecting current aliases.
	free = append(free, executionParameterField("_", ctx).Type)
	for _, alias := range ctx.importAliases {
		used[alias] = struct{}{}
	}
	for _, expression := range free {
		if expression == nil {
			continue
		}
		ast.Inspect(expression, func(node ast.Node) bool {
			if identifier, ok := node.(*ast.Ident); ok {
				used[identifier.Name] = struct{}{}
			}
			return true
		})
	}
	return used
}
