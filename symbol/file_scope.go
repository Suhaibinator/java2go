package symbol

// FileScope represents the scope in a single source file, that can contain one
// or more source classes
type FileScope struct {
	// The global package that the file is located in
	Package string
	// Every external package that is imported into the file
	// Formatted as map[ImportedType: full.package.path]
	Imports map[string]string
	// Top-level classes/interfaces/enums declared in this file, in source order
	TopLevelClasses []*ClassScope
	// The first top-level declaration (kept for backward compatibility)
	BaseClass *ClassScope
}

// FindClass searches through a file to find if a given class has been defined
// at its root class, or within any of the subclasses
func (fs *FileScope) FindClass(name string) *Definition {
	for _, top := range fs.TopLevelClasses {
		if def := top.FindClass(name); def != nil {
			return def
		}
	}
	return nil
}

// FindClassScope searches for the class scope (not just its definition) by name.
func (fs *FileScope) FindClassScope(name string) *ClassScope {
	for _, top := range fs.TopLevelClasses {
		if scope := top.FindClassScope(name); scope != nil {
			return scope
		}
	}
	return nil
}

// FindField searches through all of the classes in a file and determines if a
// field exists
func (cs *FileScope) FindField() Finder {
	cm := fileFieldFinder(*cs)
	return &cm
}

type fileFieldFinder FileScope

func findFieldsInClass(class *ClassScope, criteria func(d *Definition) bool) []*Definition {
	defs := class.FindField().By(criteria)
	for _, subclass := range class.Subclasses {
		defs = append(defs, findFieldsInClass(subclass, criteria)...)
	}
	return defs
}

func (ff *fileFieldFinder) By(criteria func(d *Definition) bool) []*Definition {
	return findFieldsInClass(ff.BaseClass, criteria)
}

func (ff *fileFieldFinder) ByName(name string) []*Definition {
	return ff.By(func(d *Definition) bool {
		return d.Name == name
	})
}

func (ff *fileFieldFinder) ByOriginalName(originalName string) []*Definition {
	return ff.By(func(d *Definition) bool {
		return d.OriginalName == originalName
	})
}
