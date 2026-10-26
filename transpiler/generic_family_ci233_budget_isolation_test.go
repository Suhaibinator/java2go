package transpiler

import (
	"bytes"
	"maps"
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func ci233GenericTypeNode(node *sitter.Node) *sitter.Node {
	if node == nil || node.Type() == "generic_type" {
		return node
	}
	for index := 0; index < int(node.NamedChildCount()); index++ {
		if found := ci233GenericTypeNode(node.NamedChild(index)); found != nil {
			return found
		}
	}
	return nil
}

func TestCI233DiscoverySharesOnlyCurrentBatchNodeFacts(t *testing.T) {
	for _, empty := range []bool{false, true} {
		name := "current-graph"
		if empty {
			name = "new-empty-batch"
		}
		t.Run(name, func(t *testing.T) {
			helper := setupParseHelper(t, `class Cell<T>{T value;} class Use{Cell<?> value;}`)
			file := helper.File.Symbols
			cell, use := file.FindClassScope("Cell"), file.FindClassScope("Use")
			if cell == nil || use == nil {
				t.Fatal("missing source declarations")
			}
			outer := &callableSubclassSourceInventory{}
			if !empty {
				foreign := &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
				// These semantic/selector records would falsely reject or bypass
				// source work if discovery borrowed them along with node facts.
				outer.graph = symbol.GlobalScope
				outer.ownership = &resolutionFileIndex{graph: foreign, files: map[*symbol.ClassScope]*symbol.FileScope{}}
				outer.ready, outer.syntheticReady, outer.selectorsReady = true, true, true
				event := callableSubclassSourceEvent{owner: use, file: file, supertype: ci233GenericTypeNode(use.Class.DeclarationNode)}
				if event.supertype == nil {
					t.Fatal("missing false-positive superclass witness")
				}
				outer.events = []callableSubclassSourceEvent{event}
				outer.syntheticEvents = []callableSubclassSourceEvent{event}
				outer.selectors = map[string]struct{}{"sentinel": {}}
				outer.staticImports = map[*symbol.FileScope]staticMethodImportSourceFacts{file: {prefix: []byte("sentinel"), ordinary: []string{"kept"}, ordinaryReady: true}}
				outer.nodes = map[*sitter.Node]resolvedSourceNodeFacts{}
			}
			before := *outer
			eventsBefore := append([]callableSubclassSourceEvent(nil), outer.events...)
			syntheticBefore := append([]callableSubclassSourceEvent(nil), outer.syntheticEvents...)
			selectorsBefore := maps.Clone(outer.selectors)
			importsBefore := maps.Clone(outer.staticImports)
			for file, facts := range importsBefore {
				facts.prefix = bytes.Clone(facts.prefix)
				facts.ordinary = append([]string(nil), facts.ordinary...)
				facts.imports = append([]staticMethodImport(nil), facts.imports...)
				importsBefore[file] = facts
			}
			var ownershipGraphBefore *symbol.GlobalSymbols
			var ownershipFilesBefore map[*symbol.ClassScope]*symbol.FileScope
			if outer.ownership != nil {
				ownershipGraphBefore = outer.ownership.graph
				ownershipFilesBefore = maps.Clone(outer.ownership.files)
			}
			ctx := helper.Ctx.Clone()
			ctx.callableSubclasses = outer
			plans := discoverCanonicalGenericFamilies(ctx)
			if len(plans) != 1 {
				t.Fatalf("discovery borrowed a caller decision: families=%d", len(plans))
			}
			if _, present := plans[0].members[cell]; !present {
				t.Fatal("source family identity changed")
			}
			if ctx.callableSubclasses != outer || outer.graph != symbol.GlobalScope || len(outer.nodes) == 0 {
				t.Fatal("current conversion batch did not retain reusable node facts")
			}
			if before.nodes != nil && len(before.nodes) == 0 {
				t.Fatal("discovery replaced the caller's node-fact map")
			}
			if outer.ownership != before.ownership || outer.ready != before.ready || outer.syntheticReady != before.syntheticReady || outer.selectorsReady != before.selectorsReady ||
				!reflect.DeepEqual(outer.events, eventsBefore) || !reflect.DeepEqual(outer.syntheticEvents, syntheticBefore) ||
				!reflect.DeepEqual(outer.selectors, selectorsBefore) || !reflect.DeepEqual(outer.staticImports, importsBefore) {
				t.Fatal("discovery published ownership, events, selectors, or imports into the caller")
			}
			if outer.ownership != nil && (outer.ownership.graph != ownershipGraphBefore ||
				!reflect.DeepEqual(outer.ownership.files, ownershipFilesBefore) ||
				outer.ownership.declarations != nil || outer.ownership.renderFile != nil || len(ownershipGraphBefore.Packages) != 0) {
				t.Fatal("discovery changed the caller's foreign ownership proof in place")
			}
		})
	}
}

func TestCI233DiscoveryDoesNotBorrowForeignNodeFacts(t *testing.T) {
	helper := setupParseHelper(t, `class Cell<T>{T value;} class Use{Cell<?> value;}`)
	root := helper.File.Symbols.FindClassScope("Use").Class.DeclarationNode
	foreign := &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	outer := &callableSubclassSourceInventory{graph: foreign, nodes: map[*sitter.Node]resolvedSourceNodeFacts{
		root: {kind: "foreign-sentinel", childrenReady: true},
	}}
	nodesBefore := maps.Clone(outer.nodes)
	ctx := helper.Ctx.Clone()
	ctx.callableSubclasses = outer
	if plans := discoverCanonicalGenericFamilies(ctx); len(plans) != 1 {
		t.Fatal("foreign node facts affected the current source family")
	}
	if outer.graph != foreign || len(outer.nodes) != 1 || !reflect.DeepEqual(outer.nodes, nodesBefore) || len(foreign.Packages) != 0 || outer.ownership != nil || outer.ready || outer.syntheticReady || outer.selectorsReady {
		t.Fatal("discovery mutated the foreign holder")
	}
}

func TestCI233ColdCallableMatchDoesNotPublishPartialInventory(t *testing.T) {
	helper := setupParseHelper(t, `class Left{} class Rght{} class Other{} class Use{Object first=new Left(){};Object second=new Rght(){};}`)
	file := helper.File.Symbols
	left, right, other := file.FindClassScope("Left"), file.FindClassScope("Rght"), file.FindClassScope("Other")
	if left == nil || right == nil || other == nil {
		t.Fatal("missing source targets")
	}
	inventory := &callableSubclassSourceInventory{graph: symbol.GlobalScope}
	ctx := helper.Ctx.Clone()
	ctx.callableSubclasses = inventory
	for _, target := range []*symbol.ClassScope{left, right} {
		if !classHasUnmodeledCallableSubclass(target, ctx) {
			t.Fatal("a distinct anonymous target was lost after an earlier match")
		}
		if inventory.ready || len(inventory.events) != 0 {
			t.Fatal("cold positive query published an incomplete event inventory")
		}
	}
	if classHasUnmodeledCallableSubclass(other, ctx) || !inventory.ready || len(inventory.events) != 2 {
		t.Fatal("complete negative traversal did not retain both source events")
	}
	index := bytes.Index(file.Source, []byte("new Left()"))
	if index < 0 {
		t.Fatal("missing mutable superclass occurrence")
	}
	copy(file.Source[index:index+len("new Left()")], []byte("new Rght()"))
	if classHasUnmodeledCallableSubclass(left, ctx) || !classHasUnmodeledCallableSubclass(right, ctx) || !inventory.ready || len(inventory.events) != 2 {
		t.Fatal("complete structural inventory cached a source target decision")
	}
}
