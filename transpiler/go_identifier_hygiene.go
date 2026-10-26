package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// Lower identifiers once the complete file exists. Names composed for static
// members, constructors, nested classes, bridges and captures then use the
// same injective mapping as their references, without changing Java lookup.
func lowerGeneratedGoIdentifiers(file *ast.File, _ Ctx) {
	seenIdentifiers := make(map[*ast.Ident]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.Ident:
			if !seenIdentifiers[value] {
				seenIdentifiers[value] = true
				value.Name = symbol.GoIdentifier(value.Name)
			}
		}
		return true
	})
}
