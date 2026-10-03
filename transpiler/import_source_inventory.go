package transpiler

import (
	"bytes"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// This scoped fallback serves import syntax only. It never supplies lexical,
// overload, selector, or resolved type facts to a fresh context.
var activeImportSourceInventory *callableSubclassSourceInventory

type staticMethodImportSourceFacts struct {
	declaration, root *sitter.Node
	sourceLength      int
	rootEnd           uint32
	rootChildren      uint32
	prefix            []byte
	imports           []staticMethodImport
	importsReady      bool
	ordinary          []string
	ordinaryReady     bool
}

func importSourceInventory(ctx Ctx) *callableSubclassSourceInventory {
	if ctx.callableSubclasses != nil {
		return resolvedSourceInventory(ctx)
	}
	inventory := activeImportSourceInventory
	if inventory == nil || inventory.graph != symbol.GlobalScope ||
		activeResolutionFiles == nil || activeResolutionFiles.graph != symbol.GlobalScope ||
		ctx.currentFile == nil || activeResolutionFiles.files[ctx.currentFile.BaseClass] != ctx.currentFile {
		return nil
	}
	return inventory
}

// A declaration and its root identify the exact parser tree. The copied prefix
// includes the header even when it has no imports; source length and structural
// bounds also invalidate appended source or an edited root. Body-only byte
// changes of equal length cannot affect these syntax tuples.
func importSourceFacts(ctx Ctx) (*callableSubclassSourceInventory, staticMethodImportSourceFacts, bool) {
	inventory := importSourceInventory(ctx)
	file := ctx.currentFile
	if inventory == nil || file == nil || file.BaseClass == nil || file.BaseClass.Class == nil {
		return nil, staticMethodImportSourceFacts{}, false
	}
	declaration := file.BaseClass.Class.DeclarationNode
	if declaration == nil {
		return nil, staticMethodImportSourceFacts{}, false
	}
	root := declaration
	for root.Parent() != nil {
		root = root.Parent()
	}
	if facts, ok := inventory.staticImports[file]; ok &&
		facts.declaration == declaration && facts.root == root &&
		facts.sourceLength == len(file.Source) && facts.rootEnd == root.EndByte() &&
		facts.rootChildren == root.NamedChildCount() && len(facts.prefix) <= len(file.Source) &&
		bytes.Equal(facts.prefix, file.Source[:len(facts.prefix)]) {
		return inventory, facts, true
	}
	end := declaration.StartByte()
	for i := 0; i < int(root.NamedChildCount()); i++ {
		node := root.NamedChild(i)
		if node.Type() == "import_declaration" && node.EndByte() > end {
			end = node.EndByte()
		}
	}
	if uint64(end) > uint64(len(file.Source)) {
		return nil, staticMethodImportSourceFacts{}, false
	}
	facts := staticMethodImportSourceFacts{
		declaration: declaration, root: root, sourceLength: len(file.Source),
		rootEnd: root.EndByte(), rootChildren: root.NamedChildCount(),
		prefix: append([]byte(nil), file.Source[:end]...),
	}
	return inventory, facts, true
}

func storeImportSourceFacts(inventory *callableSubclassSourceInventory, file *symbol.FileScope, facts staticMethodImportSourceFacts) {
	if inventory.staticImports == nil {
		inventory.staticImports = make(map[*symbol.FileScope]staticMethodImportSourceFacts)
	}
	inventory.staticImports[file] = facts
}
