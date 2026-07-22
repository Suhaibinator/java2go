package symbol

import sitter "github.com/smacker/go-tree-sitter"

// ClassScope represents a single defined class, and the declarations in it
type ClassScope struct {
	// The definition for the class defined within the class
	Class *Definition
	// Whether this class was declared as an interface.
	IsInterface bool
	// Every class that is nested within the base class
	Subclasses []*ClassScope
	// Superclass (as written in source, may include type arguments), if any.
	// Example: "Animal" or "Base<T>".
	Superclass string
	// Whether this class was declared abstract.
	IsAbstract bool
	// Any normal and static fields associated with the class
	Fields []*Definition
	// Methods and constructors
	Methods []*Definition
	// AffineArrayViews groups proven trivial accessors over the same private final
	// backing array and private final row-stride field. It is empty unless the
	// declaring class is final and every recorded invariant was proven from source.
	AffineArrayViews []*AffineArrayView
	// Whether this class is an enum
	IsEnum bool
	// Interfaces implemented by this class/enum (content as written in source)
	ImplementedInterfaces []string
	// Enum constants declared on the enum (only populated if IsEnum is true)
	EnumConstants []EnumConstant
	// Type parameters for generic classes (e.g., ["T", "U"] for class Foo<T, U>)
	TypeParameters []TypeParam
	// Whether this class contains included (non-static, non-excluded) field initializers.
	// This is set during transpilation pass and used to wire constructor initialization.
	HasInstanceFieldInitializers bool
	// IsInner is true for a non-static nested class (an "inner" class in Java
	// terms). Inner classes hold an implicit reference to an instance of their
	// enclosing class, which is modeled as a synthesized field.
	IsInner bool
	// Enclosing is the immediately-enclosing class scope for a nested class, or
	// nil for top-level classes. Set when the nested class is parsed.
	Enclosing *ClassScope
	// EnclosingField overrides the default synthesized enclosing-instance field
	// spelling when that spelling would collide with another generated selector.
	// It is primarily used by hoisted anonymous/local classes, whose Java member
	// namespaces have to be reconciled with Go's single selector namespace.
	EnclosingField string
}

// EnclosingFieldName returns the name of the synthesized field that an inner
// class uses to hold its enclosing instance (e.g. an inner class of Outer gets
// a field named "outer"). Empty for non-inner classes.
func (cs *ClassScope) EnclosingFieldName() string {
	if cs == nil || !cs.IsInner || cs.Enclosing == nil || cs.Enclosing.Class == nil {
		return ""
	}
	if cs.EnclosingField != "" {
		return cs.EnclosingField
	}
	name := Lowercase(cs.Enclosing.Class.OriginalName)
	if IsReserved(name) {
		name += "_"
	}
	return name
}

// EnumConstant represents a single enum constant and its constructor arguments.
// Arguments are stored as tree-sitter nodes so they can later be converted into
// Go expressions during code generation.
type EnumConstant struct {
	Name      string
	Arguments []*sitter.Node
	Body      *sitter.Node
}

// IsTypeParameter checks if a given name is a type parameter of this class
func (cs *ClassScope) IsTypeParameter(name string) bool {
	for _, tp := range cs.TypeParameters {
		if tp.Name == name {
			return true
		}
	}
	return false
}

func (cs *ClassScope) TypeParameterNames() []string {
	return TypeParamNames(cs.TypeParameters)
}

// FindMethod searches through the immediate class's methods find a specific method
func (cs *ClassScope) FindMethod() Finder {
	cm := classMethodFinder(*cs)
	return &cm
}

// FindField searches through the immediate class's fields to find a specific field
func (cs *ClassScope) FindField() Finder {
	cm := classFieldFinder(*cs)
	return &cm
}

type classMethodFinder ClassScope

func (cm *classMethodFinder) By(criteria func(d *Definition) bool) []*Definition {
	results := []*Definition{}
	for _, method := range cm.Methods {
		if criteria(method) {
			results = append(results, method)
		}
	}
	return results
}

func (cm *classMethodFinder) ByName(name string) []*Definition {
	return cm.By(func(d *Definition) bool {
		return d.Name == name
	})
}

func (cm *classMethodFinder) ByOriginalName(originalName string) []*Definition {
	return cm.By(func(d *Definition) bool {
		return d.OriginalName == originalName
	})
}

type classFieldFinder ClassScope

func (cm *classFieldFinder) By(criteria func(d *Definition) bool) []*Definition {
	results := []*Definition{}
	for _, method := range cm.Fields {
		if criteria(method) {
			results = append(results, method)
		}
	}
	return results
}

func (cm *classFieldFinder) ByName(name string) []*Definition {
	return cm.By(func(d *Definition) bool {
		return d.Name == name
	})
}

func (cm *classFieldFinder) ByOriginalName(originalName string) []*Definition {
	return cm.By(func(d *Definition) bool {
		return d.OriginalName == originalName
	})
}

// FindMethodByDisplayName searches for a given method by its display name
// If some ignored parameter types are specified as non-nil, it will skip over
// any function that matches these ignored parameter types exactly
func (cs *ClassScope) FindMethodByName(name string, ignoredParameterTypes []string) *Definition {
	return cs.findMethodWithComparison(func(method *Definition) bool { return method.OriginalName == name }, ignoredParameterTypes)
}

// FindMethodByDisplayName searches for a given method by its display name
// If some ignored parameter types are specified as non-nil, it will skip over
// any function that matches these ignored parameter types exactly
func (cs *ClassScope) FindMethodByDisplayName(name string, ignoredParameterTypes []string) *Definition {
	return cs.findMethodWithComparison(func(method *Definition) bool { return method.Name == name }, ignoredParameterTypes)
}

func (cs *ClassScope) findMethodWithComparison(comparison func(method *Definition) bool, ignoredParameterTypes []string) *Definition {
	for _, method := range cs.Methods {
		if comparison(method) {
			// If no parameters were specified to ignore, then return the first match
			if ignoredParameterTypes == nil {
				return method
			} else if len(method.Parameters) != len(ignoredParameterTypes) { // Size of parameters were not equal, instantly not equal
				return method
			}

			// Check the remaining paramters one-by-one
			for index, parameter := range method.Parameters {
				if parameter.OriginalType != ignoredParameterTypes[index] {
					return method
				}
			}
		}
	}

	// Not found
	return nil
}

// FindClass searches through a class file and returns the definition for the
// found class, or nil if none was found
func (cs *ClassScope) FindClass(name string) *Definition {
	if cs.Class.OriginalName == name {
		return cs.Class
	}
	for _, subclass := range cs.Subclasses {
		class := subclass.FindClass(name)
		if class != nil {
			return class
		}
	}
	return nil
}

// FindClassScope searches for the class scope by its original name.
func (cs *ClassScope) FindClassScope(name string) *ClassScope {
	if cs.Class.OriginalName == name {
		return cs
	}
	for _, subclass := range cs.Subclasses {
		if scope := subclass.FindClassScope(name); scope != nil {
			return scope
		}
	}
	return nil
}

// FindFieldByName searches for a field by its original name, and returns its definition
// or nil if none was found
func (cs *ClassScope) FindFieldByName(name string) *Definition {
	for _, field := range cs.Fields {
		if field.OriginalName == name {
			return field
		}
	}
	return nil
}

func (cs *ClassScope) FindFieldByDisplayName(name string) *Definition {
	for _, field := range cs.Fields {
		if field.Name == name {
			return field
		}
	}
	return nil
}
