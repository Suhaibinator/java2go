package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"maps"
	"testing"
)

func sourceInterfaceIsolationUse(t *testing.T, owner *symbol.ClassScope, source []byte) *sitter.Node {
	t.Helper()
	var found *sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil || found != nil {
			return
		}
		if n.Type() == "generic_type" && n.Content(source) == "Source<A>" {
			found = n
			return
		}
		for _, child := range nodeutil.NamedChildrenOf(n) {
			walk(child)
		}
	}
	walk(owner.Class.DeclarationNode)
	if found == nil {
		t.Fatal("original Source<A> occurrence missing")
	}
	return found
}
func TestGenericFamilySourceInterfaceStorageForeignOwnershipHolder(t *testing.T) {
	source := sourceStorageInterfaceFixture(t)
	h := setupParseHelper(t, source)
	target := h.File.Symbols.FindClassScope("Source")
	deferred := h.File.Symbols.FindClassScope("Deferred")
	ctx := classScopeCtx(deferred, h.Ctx)
	use := sourceInterfaceIsolationUse(t, deferred, h.File.Symbols.Source)
	graph := symbol.GlobalScope
	inventory := newGenericFamilyInventory(ctx)
	foreign := setupParseHelper(t, `class Other<X>{X value;}`)
	holder := newResolutionFileIndex()
	holder.renderFile = foreign.File.Symbols
	oldGraph, oldFiles, oldRender, oldFacts := holder.graph, maps.Clone(holder.files), holder.renderFile, holder.declarations
	symbol.GlobalScope = graph
	ctx.sourceOwnership = holder
	plan := &genericFamilyPlan{storageInventory: inventory}
	admitted := plan.queueSourceInterfaceStorage(target, "Source<A>", ctx, use)
	if holder.graph != oldGraph || !maps.Equal(holder.files, oldFiles) || holder.renderFile != oldRender || holder.declarations != oldFacts {
		t.Fatal("direct storage helper mutated foreign caller ownership holder")
	}
	if !admitted {
		t.Fatal("independent current source ownership lost valid dependency")
	}
}
func TestGenericFamilySourceInterfaceStorageForeignOccurrenceFile(t *testing.T) {
	source := sourceStorageInterfaceFixture(t)
	h := setupParseHelper(t, source)
	target := h.File.Symbols.FindClassScope("Source")
	graph := symbol.GlobalScope
	inventory := newGenericFamilyInventory(h.Ctx)
	foreign := setupParseHelper(t, source)
	owner := foreign.File.Symbols.FindClassScope("Deferred")
	ctx := classScopeCtx(owner, foreign.Ctx)
	use := sourceInterfaceIsolationUse(t, owner, foreign.File.Symbols.Source)
	symbol.GlobalScope = graph
	plan := &genericFamilyPlan{storageInventory: inventory}
	if plan.queueSourceInterfaceStorage(target, "Source<A>", ctx, use) {
		t.Fatal("unregistered foreign occurrence file acquired a current graph dependency")
	}
}
