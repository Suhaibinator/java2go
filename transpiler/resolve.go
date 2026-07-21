package transpiler

import (
	"strconv"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

// goReservedFuncNames are Go identifiers that break code generation when used
// verbatim as a generated top-level function name: `init` is reserved for
// package initialization (and may not take parameters or return a value), and
// the predeclared builtins shadow/conflict when redeclared as functions. `main`
// is intentionally excluded — a Java `public static void main(String[])` is the
// legitimate program entry point and is emitted as Go's `main`.
var goReservedFuncNames = map[string]struct{}{
	"init":    {},
	"len":     {},
	"cap":     {},
	"copy":    {},
	"new":     {},
	"make":    {},
	"append":  {},
	"delete":  {},
	"panic":   {},
	"recover": {},
	"print":   {},
	"println": {},
	"close":   {},
	"complex": {},
	"real":    {},
	"imag":    {},
}

// collidesWithGoFuncName reports whether a method's generated Go name would
// clash with a reserved/predeclared Go function name. Methods with receivers are
// safe (the name lives in the type's method set, not package scope), so this only
// applies to static methods, which are emitted as package-level functions.
func collidesWithGoFuncName(method *symbol.Definition) bool {
	if method == nil || method.Constructor || !method.IsStatic {
		return false
	}
	_, reserved := goReservedFuncNames[method.Name]
	return reserved
}

func ResolveFile(file parsing.SourceFile) {
	if file.Symbols == nil {
		return
	}
	// Complete ordinary member resolution for the entire file before allocating
	// synthesized names. A Java source may contain multiple top-level classes and
	// arbitrarily deep nested classes; helper naming must observe their final Go
	// member names, not the pre-resolution spellings.
	for _, top := range file.Symbols.TopLevelClasses {
		resolveClassTree(top, file)
	}
	// Java keeps fields and methods in separate namespaces, while Go promotion
	// gives an embedded superclass's members and the child type's direct members
	// one selector namespace. Run this package-wide after each file; on the final
	// file all ordinary names are resolved, and the pass is idempotent before
	// then because it only renames an actual remaining collision.
	resolvePromotedFieldMethodCollisions()
	resolveAffineArrayViewHelperNames(file)
}

func resolveClassTree(class *symbol.ClassScope, file parsing.SourceFile) {
	if class == nil {
		return
	}
	ResolveClass(class, file)
	for _, nested := range class.Subclasses {
		resolveClassTree(nested, file)
	}
}

func packageHasOtherStaticFieldName(packageScope *symbol.PackageScope, current *symbol.Definition, name string) bool {
	if packageScope == nil {
		return false
	}
	for _, fileScope := range packageScope.Files {
		if fileScope == nil {
			continue
		}
		for _, top := range fileScope.TopLevelClasses {
			found := false
			var visit func(*symbol.ClassScope)
			visit = func(class *symbol.ClassScope) {
				if class == nil || found {
					return
				}
				for _, field := range class.Fields {
					if field != nil && field != current && field.IsStatic && field.Name == name {
						found = true
						return
					}
				}
				for _, nested := range class.Subclasses {
					visit(nested)
				}
			}
			visit(top)
			if found {
				return true
			}
		}
	}
	return false
}

// packageHasOtherStaticMethodName reports collisions in the namespace where
// Java static methods are emitted. Java qualifies a static method by its
// declaring class, but generated Go represents it as a package-level function;
// consequently methods from different classes (including parent/child hiding)
// must have distinct generated names even when their Java signatures match.
func packageHasOtherStaticMethodName(packageScope *symbol.PackageScope, current *symbol.Definition, name string) bool {
	if packageScope == nil {
		return false
	}
	for _, fileScope := range packageScope.Files {
		if fileScope == nil {
			continue
		}
		for _, top := range fileScope.TopLevelClasses {
			found := false
			var visit func(*symbol.ClassScope)
			visit = func(class *symbol.ClassScope) {
				if class == nil || found {
					return
				}
				for _, method := range class.Methods {
					if method != nil && method != current && method.IsStatic && method.Name == name {
						found = true
						return
					}
				}
				for _, nested := range class.Subclasses {
					visit(nested)
				}
			}
			visit(top)
			if found {
				return true
			}
		}
	}
	return false
}

func classHasOtherFieldName(class *symbol.ClassScope, current *symbol.Definition, name string) bool {
	if class == nil {
		return false
	}
	for _, field := range class.Fields {
		if field != nil && field != current && field.Name == name {
			return true
		}
	}
	return false
}

func classHasOtherInstanceMethodName(class *symbol.ClassScope, current *symbol.Definition, name string) bool {
	if class == nil {
		return false
	}
	for _, method := range class.Methods {
		if method != nil && method != current && !method.IsStatic && method.Name == name {
			return true
		}
	}
	return false
}

func classHasOtherMethodName(class *symbol.ClassScope, current *symbol.Definition, name string) bool {
	if class == nil {
		return false
	}
	for _, method := range class.Methods {
		if method != nil && method != current && method.Name == name {
			return true
		}
	}
	return false
}

func classHasAffineArrayViewHelperName(class *symbol.ClassScope, name string) bool {
	if class == nil || name == "" {
		return false
	}
	for _, view := range class.AffineArrayViews {
		if view != nil && view.HelperName == name {
			return true
		}
	}
	return false
}

func classAncestorScopes(class *symbol.ClassScope) []*symbol.ClassScope {
	seen := make(map[*symbol.ClassScope]struct{})
	var ancestors []*symbol.ClassScope
	for current := class; current != nil; {
		parent := resolveSuperclassScopeInDeclaringContext(Ctx{}, current)
		if parent == nil {
			break
		}
		if _, duplicate := seen[parent]; duplicate {
			break
		}
		seen[parent] = struct{}{}
		ancestors = append(ancestors, parent)
		current = parent
	}
	return ancestors
}

func ancestorHasInheritedInstanceMethodName(ancestors []*symbol.ClassScope, name string) bool {
	for _, ancestor := range ancestors {
		for _, method := range ancestor.Methods {
			if method != nil && !method.IsStatic && !method.IsPrivate && !method.Constructor && method.Name == name {
				return true
			}
		}
	}
	return false
}

func classOrAncestorHasInterfaceDefaultMethodName(class *symbol.ClassScope, ancestors []*symbol.ClassScope, name string) bool {
	owners := make([]*symbol.ClassScope, 0, len(ancestors)+1)
	owners = append(owners, class)
	owners = append(owners, ancestors...)
	for _, owner := range owners {
		ownerFile := findFileScopeForClassScope(owner)
		ctx := Ctx{currentFile: ownerFile, currentClass: owner}
		for _, iface := range resolveImplementedInterfaceScopesInDeclaringContext(ctx, owner) {
			for _, method := range collectInterfaceDefaultMethods(iface, ctx, make(map[*symbol.ClassScope]struct{})) {
				if method != nil && method.Name == name {
					return true
				}
			}
		}
	}
	return false
}

func classHasEmbeddedInterfaceDefaultCarrierName(class *symbol.ClassScope, name string) bool {
	ownerFile := findFileScopeForClassScope(class)
	ctx := Ctx{currentFile: ownerFile, currentClass: class}
	for _, iface := range resolveImplementedInterfaceScopesInDeclaringContext(ctx, class) {
		if interfaceHasDefaultMethods(iface, ctx) && interfaceDefaultCarrierName(iface) == name {
			return true
		}
	}
	return false
}

func resolvePromotedFieldMethodCollisions() {
	for resolvePromotedFieldMethodCollisionsPass() {
	}
}

func resolvePromotedFieldMethodCollisionsPass() bool {
	changed := false
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			for _, top := range file.TopLevelClasses {
				if resolvePromotedFieldMethodCollisionsInTree(top) {
					changed = true
				}
			}
		}
	}
	return changed
}

func resolvePromotedFieldMethodCollisionsInTree(class *symbol.ClassScope) bool {
	if class == nil {
		return false
	}
	changed := false
	ancestors := classAncestorScopes(class)
	// A direct child method shadows a promoted superclass field in Go. Preserve
	// the method spelling (it may be an interface implementation or override) and
	// move the field instead; every field use resolves through its Definition.
	for _, method := range class.Methods {
		if method == nil || method.IsStatic || method.Constructor {
			continue
		}
		for _, ancestor := range ancestors {
			for _, field := range ancestor.Fields {
				if field == nil || field.IsStatic {
					continue
				}
				for suffix := 0; field.Name == method.Name ||
					classHasOtherFieldName(ancestor, field, field.Name) ||
					classHasOtherInstanceMethodName(ancestor, field, field.Name) ||
					classHasEmbeddedInterfaceDefaultCarrierName(ancestor, field.Name) ||
					classHasAffineArrayViewHelperName(ancestor, field.Name); suffix++ {
					field.Rename(field.Name + strconv.Itoa(suffix))
					changed = true
				}
			}
		}
	}
	for _, field := range class.Fields {
		if field == nil || field.IsStatic {
			continue
		}
		for suffix := 0; ancestorHasInheritedInstanceMethodName(ancestors, field.Name) ||
			classOrAncestorHasInterfaceDefaultMethodName(class, ancestors, field.Name) ||
			classHasEmbeddedInterfaceDefaultCarrierName(class, field.Name) ||
			classHasOtherFieldName(class, field, field.Name) ||
			classHasOtherInstanceMethodName(class, field, field.Name) ||
			classHasAffineArrayViewHelperName(class, field.Name); suffix++ {
			field.Rename(field.Name + strconv.Itoa(suffix))
			changed = true
		}
	}
	for _, method := range class.Methods {
		if method == nil || method.IsStatic || method.Constructor {
			continue
		}
		for suffix := 0; classHasEmbeddedInterfaceDefaultCarrierName(class, method.Name) ||
			classHasOtherFieldName(class, method, method.Name) ||
			classHasOtherMethodName(class, method, method.Name) ||
			classHasAffineArrayViewHelperName(class, method.Name); suffix++ {
			method.Rename(method.Name + strconv.Itoa(suffix))
			changed = true
		}
	}
	for _, nested := range class.Subclasses {
		if resolvePromotedFieldMethodCollisionsInTree(nested) {
			changed = true
		}
	}
	return changed
}

func ResolveClass(class *symbol.ClassScope, file parsing.SourceFile) {
	packageScope := symbol.GlobalScope.FindPackage(file.Symbols.Package)

	// Resolve all the fields in that respective class
	for _, field := range class.Fields {

		// Since a private global variable is able to be accessed in the package, it must be renamed
		// to avoid conflicts with other global variables

		symbol.ResolveDefinition(field, file.Symbols)

		// Rename the field if its name conflits with any keyword
		for i := 0; symbol.IsReserved(field.Name) ||
			classHasOtherFieldName(class, field, field.Name) ||
			(!field.IsStatic && classHasOtherInstanceMethodName(class, field, field.Name)) ||
			(field.IsStatic && packageHasOtherStaticFieldName(packageScope, field, field.Name)) ||
			(field.IsStatic && packageHasOtherStaticMethodName(packageScope, field, field.Name)); i++ {
			field.Rename(field.Name + strconv.Itoa(i))
		}
	}

	// Resolve all the methods
	for _, method := range class.Methods {
		// Resolve the return type, as well as the body of the method
		symbol.ResolveChildren(method, file.Symbols)

		// Comparison compares the method against the found method
		// This tests for a method of the same name, but with different
		// aspects of it, so that it can be identified as a duplicate
		comparison := func(d *symbol.Definition) bool {
			// The names must match, but everything else must be different
			if method.Name != d.Name {
				return false
			}

			// Size of parameters do not match
			if len(method.Parameters) != len(d.Parameters) {
				return true
			}

			// Go through the types and check to see if they differ
			for index, param := range method.Parameters {
				if param.OriginalType != d.Parameters[index].OriginalType {
					return true
				}
			}

			// Both methods are equal, skip this method since it is likely
			// the same method that we are trying to find duplicates of
			return false
		}

		for i := 0; symbol.IsReserved(method.Name) ||
			collidesWithGoFuncName(method) ||
			classHasOtherFieldName(class, method, method.Name) ||
			(method.IsStatic && packageHasOtherStaticFieldName(packageScope, method, method.Name)) ||
			(method.IsStatic && packageHasOtherStaticMethodName(packageScope, method, method.Name)) ||
			len(class.FindMethod().By(comparison)) > 0; i++ {
			method.Rename(method.Name + strconv.Itoa(i))
		}
		// Resolve all the paramters of the method
		for _, param := range method.Parameters {
			symbol.ResolveDefinition(param, file.Symbols)

			for i := 0; symbol.IsReserved(param.Name); i++ {
				param.Rename(param.Name + strconv.Itoa(i))
			}
		}
	}
}
