package symbol

// Definition represents the name and type of a single symbol
type Definition struct {
	// The original Java name
	OriginalName string
	// The display name of the definition, may be different from the original name
	Name string
	// Original Java type of the object
	OriginalType string
	// Display type of the object
	Type string
	// Type parameters declared on this definition (methods/constructors)
	TypeParameters []TypeParam
	// Whether this definition is static (applies to methods/fields)
	IsStatic bool
	// Indicates that this definition requires a helper to model method-level type parameters
	RequiresHelper bool
	// Name of the helper type to use (if RequiresHelper)
	HelperName string

	// If the definition is a constructor
	// This is used so that the definition handles its special naming and
	// type rules correctly
	Constructor bool
	// If the object is a function, it has parameters
	Parameters []*Definition
	// Children of the declaration, if the declaration is a scope
	Children []*Definition
}

// Rename changes the display name of a definition
func (d *Definition) Rename(name string) {
	d.Name = name
}

// ParameterByName returns a parameter's definition, given its original name
func (d *Definition) ParameterByName(name string) *Definition {
	for _, param := range d.Parameters {
		if param.OriginalName == name {
			return param
		}
	}
	return nil
}

// OriginalParameterTypes returns a list of the original types for all the parameters
func (d *Definition) OriginalParameterTypes() []string {
	names := make([]string, len(d.Parameters))
	for ind, param := range d.Parameters {
		names[ind] = param.OriginalType
	}
	return names
}

// FindVariable searches a definition's immediate children and parameters
// to try and find a given variable by its original name
func (d *Definition) FindVariable(name string) *Definition {
	for _, param := range d.Parameters {
		if param.OriginalName == name {
			return param
		}
	}
	for _, child := range d.Children {
		if child.OriginalName == name {
			return child
		}
	}
	return nil
}

func (d Definition) IsEmpty() bool {
	return d.OriginalName == "" && len(d.Children) == 0
}

func (d *Definition) TypeParameterNames() []string {
	if d == nil {
		return nil
	}
	return TypeParamNames(d.TypeParameters)
}
