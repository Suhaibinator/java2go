package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Record only source syntax and its declaring owner/file. Named classes are
// included here because the original synthetic-subclass query also observes
// them. Resolution and the active local-class registry remain query-specific.
// The caller's graph-scoped inventory resets these events with all other source
// facts; callers without an inventory retain the original traversal/early exit.
func visitSyntheticSuperclassSources(visit func(*symbol.ClassScope, *symbol.FileScope, *sitter.Node) bool, ctx Ctx) bool {
	for _, owner := range allSourceClassScopes() {
		if owner == nil || owner.Class == nil || owner.Class.DeclarationNode == nil {
			continue
		}
		file := findFileScopeForClassScope(owner, ctx)
		if file == nil {
			continue
		}
		var walk func(*sitter.Node) bool
		walk = func(node *sitter.Node) bool {
			if node == nil {
				return false
			}
			var supertype *sitter.Node
			switch resolvedSourceNodeType(node, ctx) {
			case "object_creation_expression":
				for _, child := range resolvedSourceNamedChildren(node, ctx) {
					if resolvedSourceNodeType(child, ctx) == "class_body" {
						supertype = node.ChildByFieldName("type")
						break
					}
				}
			case "class_declaration":
				if superclass := node.ChildByFieldName("superclass"); superclass != nil {
					types := collectTypeNodes(superclass)
					if len(types) > 0 {
						supertype = types[0]
					}
				}
			}
			if supertype != nil && visit(owner, file, supertype) {
				return true
			}
			for _, child := range resolvedSourceNamedChildren(node, ctx) {
				if walk(child) {
					return true
				}
			}
			return false
		}
		if walk(owner.Class.DeclarationNode) {
			return true
		}
	}
	return false
}
