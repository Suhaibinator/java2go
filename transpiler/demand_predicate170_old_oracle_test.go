package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"sort"
)

func demandPredicate170OldSeeds(ctx Ctx) []*symbol.ClassScope {
	seen := map[*symbol.ClassScope]bool{}
	var seeds []*symbol.ClassScope
	for _, owner := range allSourceClassScopes() {
		if owner.Class == nil || owner.Class.DeclarationNode == nil {
			continue
		}
		file := findFileScopeForClassScope(owner, ctx)
		if file == nil {
			continue
		}
		root := owner.Class.DeclarationNode
		var walk func(*sitter.Node, Ctx)
		walk = func(node *sitter.Node, lexical Ctx) {
			if node == nil {
				return
			}
			if !node.Equal(root) && sourceGenericViewTypeDeclaration(node, ctx) {
				// Named classes are visited in their own declaration context. Local
				// class scopes are not in the resolved graph; do not resolve their
				// private binders or nested type names as enclosing declarations.
				return
			}
			if resolvedSourceNodeType(node, ctx) == "class_body" {
				parent := node.Parent()
				if parent != nil && (resolvedSourceNodeType(parent, ctx) == "object_creation_expression" || resolvedSourceNodeType(parent, ctx) == "enum_constant") {
					// Anonymous declarations are hoisted later. Their lexical type
					// namespace is not this resolved owner; do not invent a demand
					// by resolving an anonymous member's shadow as an outer type.
					return
				}
			}
			if resolvedSourceNodeType(node, ctx) == "method_declaration" || resolvedSourceNodeType(node, ctx) == "constructor_declaration" {
				lexical = lexical.Clone()
				lexical.localScope = nil
				for _, method := range owner.Methods {
					if method.DeclarationNode != nil && method.DeclarationNode.Equal(node) {
						lexical.localScope = method
						break
					}
				}
			}
			if seed := sourceGenericViewDemandType(node, lexical, file.Source); seed != nil && !seen[seed] {
				seen[seed] = true
				seeds = append(seeds, seed)
			}
			for _, child := range resolvedSourceNamedChildren(node, ctx) {
				childCtx := lexical
				// The original helper returns this context for non-declarations.
				// Reuse its structural guard while keeping header/body selection fresh.
				if sourceGenericViewTypeDeclaration(node, ctx) {
					childCtx = classDeclarationChildCtx(node, child, lexical)
				}
				walk(child, childCtx)
			}
		}
		walk(root, classScopeCtx(owner, ctx))
	}
	// The project symbol graph is map-backed. Declaration identity selects the
	// layout; sorting prevents file-render order from selecting a different seed.
	sort.Slice(seeds, func(i, j int) bool {
		return qualifiedSourceClassName(seeds[i]) < qualifiedSourceClassName(seeds[j])
	})
	return seeds
}
