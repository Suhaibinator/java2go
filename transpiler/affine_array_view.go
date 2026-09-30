package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

const affineArrayViewHelperPrefix = "Java2goAffineView"

// resolveAffineArrayViewHelperNames assigns exported method names after normal
// member resolution. The numeric discriminator sits inside the name, and the
// collision check also reserves names that ordinary resolution could produce by
// appending digits in a different file resolved later.
func resolveAffineArrayViewHelperNames(file parsing.SourceFile) {
	if file.Symbols == nil {
		return
	}
	for _, top := range file.Symbols.TopLevelClasses {
		resolveAffineArrayViewHelperNamesInClass(top, file.Symbols)
	}
}

func resolveAffineArrayViewHelperNamesInClass(class *symbol.ClassScope, file *symbol.FileScope) {
	if class == nil {
		return
	}
	if len(class.AffineArrayViews) > 0 {
		reserved := make(map[string]struct{})
		collectAffineArrayViewReservedNames(class, file, reserved, make(map[*symbol.ClassScope]struct{}))

		for _, view := range class.AffineArrayViews {
			if view == nil || view.ArrayField == nil {
				continue
			}
			if view.HelperName != "" {
				reserved[view.HelperName] = struct{}{}
				continue
			}
			fieldPart := symbol.Uppercase(sanitizeGoIdent(view.ArrayField.Name))
			for discriminator := 0; ; discriminator++ {
				candidate := affineArrayViewHelperPrefix + strconv.Itoa(discriminator) + fieldPart
				if affineArrayViewHelperNameCollides(candidate, reserved) {
					continue
				}
				view.HelperName = candidate
				reserved[candidate] = struct{}{}
				break
			}
		}
	}

	for _, nested := range class.Subclasses {
		resolveAffineArrayViewHelperNamesInClass(nested, file)
	}
}

func affineArrayViewHelperNameCollides(candidate string, reserved map[string]struct{}) bool {
	for name := range reserved {
		if name == candidate {
			return true
		}
		if name == "" || !strings.HasPrefix(candidate, name) || len(candidate) == len(name) {
			continue
		}
		// ResolveClass disambiguates fields and overloads by appending decimal
		// digits. Treat every such possible future spelling as occupied because
		// another file's members may not have been resolved yet.
		digitsOnly := true
		for _, char := range candidate[len(name):] {
			if char < '0' || char > '9' {
				digitsOnly = false
				break
			}
		}
		if digitsOnly {
			return true
		}
	}
	return false
}

func affineArrayViewDeclaringFile(class *symbol.ClassScope, fallback *symbol.FileScope) *symbol.FileScope {
	contains := func(file *symbol.FileScope) bool {
		if file == nil {
			return false
		}
		for _, top := range file.TopLevelClasses {
			if classScopeContains(top, class) {
				return true
			}
		}
		return false
	}
	if contains(fallback) {
		return fallback
	}
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if contains(file) {
				return file
			}
		}
	}
	return fallback
}

func collectAffineArrayViewReservedNames(class *symbol.ClassScope, declaringFile *symbol.FileScope, reserved map[string]struct{}, seen map[*symbol.ClassScope]struct{}) {
	if class == nil {
		return
	}
	if _, duplicate := seen[class]; duplicate {
		return
	}
	seen[class] = struct{}{}
	declaringFile = affineArrayViewDeclaringFile(class, declaringFile)
	ctx := Ctx{currentFile: declaringFile, currentClass: class}

	if class.Class != nil {
		reserved[class.Class.Name] = struct{}{}
		reserved[classSelfSetterName(class)] = struct{}{}
		reserved[classDispatchFieldName(class)] = struct{}{}
	}
	reserved[fieldInitMethodName] = struct{}{}
	reserved[class.EnclosingFieldName()] = struct{}{}
	reserved["ThrowableTypeName"] = struct{}{}
	for _, field := range class.Fields {
		if field != nil {
			reserved[field.Name] = struct{}{}
		}
	}
	for _, method := range class.Methods {
		if method != nil {
			reserved[method.Name] = struct{}{}
		}
	}
	for _, view := range class.AffineArrayViews {
		if view != nil && view.HelperName != "" {
			reserved[view.HelperName] = struct{}{}
		}
	}

	if parent := resolveSuperclassScope(ctx, class); parent != nil {
		collectAffineArrayViewReservedNames(parent, affineArrayViewDeclaringFile(parent, nil), reserved, seen)
	}
	for _, implemented := range class.ImplementedInterfaces {
		base, _ := parseJavaTypeString(implemented)
		if iface := resolveClassScopeByQualifiedName(ctx, base); iface != nil {
			// A Java interface with defaults is represented by an anonymously
			// embedded carrier whose promoted selector uses this synthesized name.
			// Reserving it unconditionally is harmless and avoids coupling naming to
			// whether a transitive default was already resolved.
			reserved[interfaceDefaultCarrierName(iface)] = struct{}{}
			collectAffineArrayViewReservedNames(iface, affineArrayViewDeclaringFile(iface, nil), reserved, seen)
		}
	}
}

// generateAffineArrayViewDecls emits one nil-safe helper per proven backing
// view. Calling the helper before a zero-trip loop must not introduce an eager
// nil-receiver panic. A later call-site rewrite must still preserve Java's
// NullPointerException timing explicitly; indexing the returned nil slice would
// otherwise be classified as ArrayIndexOutOfBoundsException.
func generateAffineArrayViewDecls(ctx Ctx) []ast.Decl {
	class := ctx.currentClass
	if class == nil || class.Class == nil || len(class.AffineArrayViews) == 0 {
		return nil
	}

	receiverName := ShortName(class.Class.Name)
	receiverType := &ast.StarExpr{X: &ast.Ident{Name: class.Class.Name}}
	declarations := make([]ast.Decl, 0, len(class.AffineArrayViews))
	for _, view := range class.AffineArrayViews {
		if view == nil || view.HelperName == "" || view.ArrayField == nil || view.SizeField == nil {
			continue
		}
		arrayType := javaTypeStringToGoTypeExpr(view.ArrayField.OriginalType, nil, ctx)
		arrayValue := ast.Expr(&ast.SelectorExpr{X: &ast.Ident{Name: receiverName}, Sel: &ast.Ident{Name: view.ArrayField.Name}})
		if component, primitive := javaPrimitiveArrayComponent(view.ArrayField.OriginalType); primitive {
			componentType := javaTypeStringToGoTypeExpr(component, nil, ctx)
			arrayType = &ast.ArrayType{Elt: componentType}
			arrayValue = stdjavaCall(ctx, "PrimitiveArrayElements", arrayValue)
		}
		declarations = append(declarations, &ast.FuncDecl{
			Name: &ast.Ident{Name: view.HelperName},
			Recv: &ast.FieldList{List: []*ast.Field{{
				Names: []*ast.Ident{{Name: receiverName}},
				Type:  receiverType,
			}}},
			Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{
				{Type: arrayType},
				{Type: &ast.Ident{Name: "int32"}},
			}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.IfStmt{
					Cond: &ast.BinaryExpr{
						X:  &ast.Ident{Name: receiverName},
						Op: token.EQL,
						Y:  &ast.Ident{Name: "nil"},
					},
					Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{
						&ast.Ident{Name: "nil"},
						&ast.BasicLit{Kind: token.INT, Value: "0"},
					}}}},
				},
				&ast.ReturnStmt{Results: []ast.Expr{
					arrayValue,
					&ast.SelectorExpr{X: &ast.Ident{Name: receiverName}, Sel: &ast.Ident{Name: view.SizeField.Name}},
				}},
			}},
		})
	}
	return declarations
}
