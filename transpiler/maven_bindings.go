package transpiler

import (
	"go/ast"
	"go/types"
)

// The type checker can skip subexpressions whose surrounding type is invalid
// during a Java package cycle (for example imported.Map{Key: 1}). Recover their
// lexical references from the scopes it built, without guessing from spelling.
// Literal identifier keys stay deferred until the merged types distinguish
// struct fields from map keys and slice/array indices.
func projectLexicalBindings(file *ast.File, info *types.Info) (map[*ast.Ident]types.Object, map[*ast.Ident]bool) {
	nonReferences := map[*ast.Ident]bool{file.Name: true}
	keys := map[*ast.Ident]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.SelectorExpr:
			nonReferences[value.Sel] = true
		case *ast.Field:
			for _, name := range value.Names {
				nonReferences[name] = true
			}
		case *ast.FuncDecl:
			nonReferences[value.Name] = true
		case *ast.TypeSpec:
			nonReferences[value.Name] = true
		case *ast.ValueSpec:
			for _, name := range value.Names {
				nonReferences[name] = true
			}
		case *ast.ImportSpec:
			nonReferences[value.Name] = true
		case *ast.LabeledStmt:
			nonReferences[value.Label] = true
		case *ast.BranchStmt:
			nonReferences[value.Label] = true
		case *ast.KeyValueExpr:
			if key, ok := value.Key.(*ast.Ident); ok {
				keys[key] = true
			}
		}
		return true
	})
	objects := map[*ast.Ident]types.Object{}
	ast.Inspect(file, func(node ast.Node) bool {
		ident, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		obj := info.Uses[ident]
		if obj == nil {
			obj = info.Defs[ident]
		}
		if obj == nil && !nonReferences[ident] {
			if scope := info.Scopes[file]; scope != nil {
				if scope = scope.Innermost(ident.Pos()); scope != nil {
					_, obj = scope.LookupParent(ident.Name, ident.Pos())
				}
			}
		}
		objects[ident] = obj
		return true
	})
	return objects, keys
}
