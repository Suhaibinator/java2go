package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// Declaration ownership is constant for a registered resolved source graph.
// Resolution installs a temporary index; an explicit conversion inventory may
// own another index for its batch. Unindexed synthesized classes retain ordinary
// ownership fallback, and no index crosses a conversion's graph identity.
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

// sourceOwnershipIndex carries only declaration ownership. A fresh matcher must
// still resolve names in its own file/class context on every query. Registered
// graph replacement resets this shared facts holder; late synthesized classes
// remain unindexed and use the ordinary ownership search.
func sourceOwnershipIndex(ctx Ctx) *resolutionFileIndex {
	if index := ctx.sourceOwnership; index != nil {
		if index.graph != symbol.GlobalScope {
			*index = *newResolutionFileIndex()
		}
		return index
	}
	inventory := resolvedSourceInventory(ctx)
	if inventory == nil {
		return nil
	}
	if inventory.ownership == nil {
		inventory.ownership = newResolutionFileIndex()
	}
	return inventory.ownership
}
