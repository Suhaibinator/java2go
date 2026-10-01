package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// Declaration ownership is constant while resolving a registered source graph.
// Keep this index only for that phase: later synthesized local classes continue
// to use the ordinary ownership fallback, and another conversion cannot reuse
// entries belonging to a previous graph.
type resolutionFileIndex struct {
	graph *symbol.GlobalSymbols
	files map[*symbol.ClassScope]*symbol.FileScope
}

var activeResolutionFiles *resolutionFileIndex

func newResolutionFileIndex() *resolutionFileIndex {
	index := &resolutionFileIndex{graph: symbol.GlobalScope, files: make(map[*symbol.ClassScope]*symbol.FileScope)}
	for _, pkg := range index.graph.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			var visit func(*symbol.ClassScope)
			visit = func(scope *symbol.ClassScope) {
				if scope == nil || index.files[scope] != nil {
					return
				}
				index.files[scope] = file
				for _, nested := range scope.Subclasses {
					visit(nested)
				}
			}
			for _, top := range file.TopLevelClasses {
				visit(top)
			}
		}
	}
	return index
}
