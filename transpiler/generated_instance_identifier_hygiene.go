package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Synthetic receivers and allocated instances share Go's type/value namespace.
// Reserve the declaring package's emitted types as well as source bindings and
// hidden formals before allocating names. This never changes Java symbol data.
func generatedInstanceUsedNames(ctx Ctx, fields ...*ast.Field) map[string]struct{} {
	var declaration *sitter.Node
	if ctx.currentFile != nil && ctx.currentClass != nil && ctx.currentClass.Class != nil {
		declaration = ctx.currentClass.Class.DeclarationNode
	}
	return generatedInstanceUsedNamesForDeclaration(ctx, declaration, fields...)
}

// The caller selects the declaration whose lexical names are in scope. Keep
// package types, source bindings, and final signature reservations shared.
func generatedInstanceUsedNamesForDeclaration(ctx Ctx, declaration *sitter.Node, fields ...*ast.Field) map[string]struct{} {
	var source []byte
	if ctx.currentFile != nil {
		source = ctx.currentFile.Source
	} else {
		declaration = nil
	}
	used := affineLoopUsedNames(declaration, source, ctx)
	// Both users emit allocation; clone also emits null checks and interface types.
	for _, name := range []string{"new", "nil", "interface"} {
		used[name] = struct{}{}
	}
	var reserve func(*symbol.ClassScope)
	reserve = func(scope *symbol.ClassScope) {
		if scope == nil || scope.Class == nil {
			return
		}
		for _, name := range []string{scope.Class.Name, scope.Class.Name + "I", classDispatchTypeName(scope)} {
			if name != "" {
				used[name] = struct{}{}
			}
		}
		for _, child := range scope.Subclasses {
			reserve(child)
		}
	}
	// Direct package ownership avoids a repeated whole-program class search.
	if ctx.currentFile != nil {
		if pkg := symbol.GlobalScope.Packages[ctx.currentFile.Package]; pkg != nil {
			for _, file := range pkg.Files {
				if file != nil {
					for _, scope := range file.TopLevelClasses {
						reserve(scope)
					}
				}
			}
		} else {
			for _, scope := range ctx.currentFile.TopLevelClasses {
				reserve(scope)
			}
		}
	}
	// Hoisted local/anonymous owners may not be in the package inventory yet.
	reserve(ctx.currentClass)
	if ctx.className != "" {
		used[ctx.className] = struct{}{}
	}
	for _, parameter := range visibleTypeParameterDeclarations(ctx) {
		used[parameter.EmittedName()] = struct{}{}
	}
	if ctx.currentClass != nil {
		for _, name := range ctx.currentClass.GoTypeParameterNames() {
			used[name] = struct{}{}
		}
	}
	for _, field := range fields {
		if field == nil {
			continue
		}
		for _, name := range field.Names {
			if name != nil {
				used[name.Name] = struct{}{}
			}
		}
		// Type expressions can contain late emitted witnesses and binder spellings.
		// A selected foreign type's package alias is a binding; its selector is not.
		var reserveType func(ast.Expr)
		reserveType = func(expr ast.Expr) {
			switch typ := expr.(type) {
			case *ast.Ident:
				used[typ.Name] = struct{}{}
			case *ast.SelectorExpr:
				reserveType(typ.X)
			case *ast.StarExpr:
				reserveType(typ.X)
			case *ast.IndexExpr:
				reserveType(typ.X)
				reserveType(typ.Index)
			case *ast.IndexListExpr:
				reserveType(typ.X)
				for _, arg := range typ.Indices {
					reserveType(arg)
				}
			case *ast.ArrayType:
				reserveType(typ.Elt)
			case *ast.MapType:
				reserveType(typ.Key)
				reserveType(typ.Value)
			case *ast.ChanType:
				reserveType(typ.Value)
			case *ast.Ellipsis:
				reserveType(typ.Elt)
			case *ast.ParenExpr:
				reserveType(typ.X)
			}
		}
		reserveType(field.Type)
	}
	return used
}
