package transpiler

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/NickyBoy89/java2go/symbol"
)

// Every Java allocation has distinct identity, including stateless objects.
// Go may coalesce pointers to zero-sized structs. A blank pointer supplies
// storage without allocating a second object or depending on a predeclared Go
// type name, which a source declaration or generic binder could shadow.
func fieldsWithAllocationIdentity(structName string, fields *ast.FieldList, parameters []symbol.TypeParam, ctx Ctx) *ast.FieldList {
	builtinVisible := func(name string) bool {
		if name == structName {
			return false
		}
		for _, parameter := range parameters {
			if parameter.EmittedName() == name {
				return false
			}
		}
		if ctx.currentFile != nil {
			if fileScopeDeclaresGoIdent(ctx.currentFile, name) {
				return false
			}
			if pkg := symbol.GlobalScope.FindPackage(ctx.currentFile.Package); pkg != nil {
				for _, file := range pkg.Files {
					if file != ctx.currentFile && fileScopeDeclaresGoIdent(file, name) {
						return false
					}
				}
			}
		}
		return true
	}
	if fields != nil {
		for _, field := range fields.List {
			if generatedTypeHasStorage(field.Type, builtinVisible) {
				return fields
			}
		}
	}
	result := &ast.FieldList{}
	if fields != nil {
		*result = *fields
		result.List = append([]*ast.Field(nil), fields.List...)
	}
	result.List = append(result.List, &ast.Field{
		Names: []*ast.Ident{ast.NewIdent("_")},
		Type:  &ast.StarExpr{X: &ast.StructType{Fields: &ast.FieldList{}}},
	})
	return result
}

// Structural Go types supply a size proof directly. Only an unshadowed Go
// universe type can supply a named proof; source aliases and instantiations
// remain unknown, including fixed arrays of possibly zero-sized elements.
func generatedTypeHasStorage(expression ast.Expr, builtinVisible func(string) bool) bool {
	switch typ := expression.(type) {
	case *ast.Ident:
		if !builtinVisible(typ.Name) {
			return false
		}
		name, ok := types.Universe.Lookup(typ.Name).(*types.TypeName)
		if !ok {
			return false
		}
		switch underlying := name.Type().Underlying().(type) {
		case *types.Basic:
			return underlying.Kind() != types.Invalid
		case *types.Interface:
			return underlying.IsMethodSet()
		}
		return false
	case *ast.StarExpr, *ast.InterfaceType, *ast.FuncType, *ast.MapType, *ast.ChanType:
		return true
	case *ast.ParenExpr:
		return generatedTypeHasStorage(typ.X, builtinVisible)
	case *ast.ArrayType:
		if typ.Len == nil {
			return true
		} // slices have a nonzero header
		length, ok := typ.Len.(*ast.BasicLit)
		if !ok || length.Kind != token.INT {
			return false
		}
		value := constant.MakeFromLiteral(length.Value, token.INT, 0)
		return value.Kind() == constant.Int && constant.Sign(value) > 0 && generatedTypeHasStorage(typ.Elt, builtinVisible)
	case *ast.StructType:
		if typ.Fields != nil {
			for _, field := range typ.Fields.List {
				if generatedTypeHasStorage(field.Type, builtinVisible) {
					return true
				}
			}
		}
	}
	return false
}
