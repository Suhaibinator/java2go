package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Java permits a local and a class-qualified static field to have the same
// name. Go has no qualifier for a variable in its own package. Address helpers
// preserve the backing variable (and lvalue semantics) across that boundary;
// they deliberately perform no class initialization themselves.
func staticFieldStorageHelperName(field *symbol.Definition, owner *symbol.ClassScope) string {
	// Length-delimit the field name so a retry suffix cannot collide with a
	// different field whose name already ends in that suffix.
	base := "Java2goStaticStorage" + strconv.Itoa(len(field.Name)) + "_" + field.Name
	for suffix := 0; ; suffix++ {
		candidate := base
		if suffix > 0 {
			candidate += "_" + strconv.Itoa(suffix)
		}
		if generatedIdentifierExists(candidate, owner) {
			continue
		}
		occupied := false
		for _, pkg := range symbol.GlobalScope.Packages {
			for _, file := range pkg.Files {
				// A conservative source-byte match also catches block, catch, lambda,
				// anonymous-class and initializer bindings not attached to method scopes.
				if strings.Contains(string(file.Source), candidate) {
					occupied = true
					break
				}
				for _, top := range file.TopLevelClasses {
					if visitClassScopes(top, func(scope *symbol.ClassScope) bool {
						if scope.Class != nil && ShortName(scope.Class.Name) == candidate {
							return true
						}
						for _, method := range scope.Methods {
							if definitionBindsGoName(method, candidate) {
								return true
							}
						}
						return false
					}) {
						occupied = true
						break
					}
				}
				if occupied {
					break
				}
			}
			if occupied {
				break
			}
		}
		if !occupied {
			return candidate
		}
	}
}

func definitionBindsGoName(def *symbol.Definition, name string) bool {
	if def == nil {
		return false
	}
	if def.Name == name {
		return true
	}
	for _, param := range def.Parameters {
		if definitionBindsGoName(param, name) {
			return true
		}
	}
	for _, child := range def.Children {
		if definitionBindsGoName(child, name) {
			return true
		}
	}
	return false
}

func staticFieldStorageShadowed(field *symbol.Definition, ctx Ctx) bool {
	if definitionBindsGoName(ctx.localScope, field.Name) {
		return true
	}
	return ctx.currentClass != nil && ctx.currentClass.Class != nil && ShortName(ctx.currentClass.Class.Name) == field.Name
}

// Build from the actual emitted ValueSpec, so the pointer type is exactly the
// backing storage ABI, including interface and generic-field representations.
func staticFieldStorageHelperDecls(variables *ast.GenDecl, ctx Ctx) []ast.Decl {
	var declarations []ast.Decl
	for _, spec := range variables.Specs {
		value, ok := spec.(*ast.ValueSpec)
		if !ok || value.Type == nil {
			continue
		}
		for _, name := range value.Names {
			var field *symbol.Definition
			for _, candidate := range ctx.currentClass.Fields {
				if candidate.IsStatic && candidate.Name == name.Name {
					field = candidate
					break
				}
			}
			if field == nil {
				continue
			}
			declarations = append(declarations, &ast.FuncDecl{
				Name: ast.NewIdent(staticFieldStorageHelperName(field, ctx.currentClass)),
				Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: value.Type}}}}},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: ast.NewIdent(name.Name)}}}}},
			})
		}
	}
	return declarations
}
