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
	ResolveClass(file.Symbols.BaseClass, file)
	for _, subclass := range file.Symbols.BaseClass.Subclasses {
		ResolveClass(subclass, file)
	}
}

func ResolveClass(class *symbol.ClassScope, file parsing.SourceFile) {
	// Resolve all the fields in that respective class
	for _, field := range class.Fields {

		// Since a private global variable is able to be accessed in the package, it must be renamed
		// to avoid conflicts with other global variables

		packageScope := symbol.GlobalScope.FindPackage(file.Symbols.Package)

		symbol.ResolveDefinition(field, file.Symbols)

		// Rename the field if its name conflits with any keyword
		for i := 0; symbol.IsReserved(field.Name) ||
			len(packageScope.ExcludeFile(class.Class.Name).FindStaticField().ByName(field.Name)) > 0; i++ {
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

		for i := 0; symbol.IsReserved(method.Name) || collidesWithGoFuncName(method) || len(class.FindMethod().By(comparison)) > 0; i++ {
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
