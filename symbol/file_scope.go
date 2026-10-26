package symbol

import sitter "github.com/smacker/go-tree-sitter"

// FileScope represents the scope in a single source file, that can contain one
// or more source classes
type FileScope struct {
	// parsedRoot is the original tree passed to ParseSymbols. Keep it independent
	// of mutable class/definition aliases; only the parsing owner initializes it.
	parsedRoot *sitter.Node
	// sourceInput is bound once to this exact file and its retained parse anchor.
	sourceInput *SourceInputSnapshot
	// SourceFilename records the name associated with the retained source input.
	SourceFilename string
	// Source retains the parsed Java bytes so later whole-program symbol passes
	// can classify declarations that refer to members in another source file.
	Source []byte
	// OriginalSource/SourceOffsets are present only when grammar-equivalent
	// Unicode escape canonicalization changed byte positions. Offsets map each
	// parser-source byte (and EOF) back to the unchanged original input.
	OriginalSource []byte
	SourceOffsets  []uint32
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

// ParsedRoot returns this file's original parse anchor. Synthetic or zero-value
// file scopes have no parse certificate. Callers cannot replace the anchor.
func (fs *FileScope) ParsedRoot() *sitter.Node {
	if fs == nil {
		return nil
	}
	return fs.parsedRoot
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
