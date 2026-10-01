package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Facts are source syntax only. Each Node retains its Tree; the tree-sitter
// binding interns Node pointers within that tree. Never key by a source name or
// byte offset: separate trees may contain identical syntax at identical offsets.
// Child slices are read-only and retain the original named-child source order.
type resolvedSourceNodeFacts struct {
	kind          string
	children      []*sitter.Node
	childrenReady bool
}

func resolvedSourceInventory(ctx Ctx) *callableSubclassSourceInventory {
	inventory := ctx.callableSubclasses
	if inventory != nil && inventory.graph != symbol.GlobalScope {
		// All facts have the same graph lifetime, including negative/empty syntax.
		*inventory = callableSubclassSourceInventory{graph: symbol.GlobalScope}
	}
	return inventory
}

func resolvedSourceNodeType(node *sitter.Node, ctx Ctx) string {
	if node == nil {
		return ""
	}
	inventory := resolvedSourceInventory(ctx)
	if inventory == nil {
		return node.Type()
	}
	if inventory.nodes == nil {
		inventory.nodes = make(map[*sitter.Node]resolvedSourceNodeFacts)
	}
	facts := inventory.nodes[node]
	if facts.kind == "" {
		facts.kind = node.Type()
		inventory.nodes[node] = facts
	}
	return facts.kind
}

func resolvedSourceNamedChildren(node *sitter.Node, ctx Ctx) []*sitter.Node {
	inventory := resolvedSourceInventory(ctx)
	if inventory == nil {
		return nodeutil.NamedChildrenOf(node)
	}
	if inventory.nodes == nil {
		inventory.nodes = make(map[*sitter.Node]resolvedSourceNodeFacts)
	}
	facts := inventory.nodes[node]
	if !facts.childrenReady {
		facts.children = nodeutil.NamedChildrenOf(node)
		facts.childrenReady = true
		inventory.nodes[node] = facts
	}
	return facts.children
}
