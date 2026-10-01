package transpiler

import (
	"sort"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Source generic casts and raw/wildcard references can observe different Java
// views of the same object. Discover those demands before choosing any physical
// layout. This inventory uses only lexical source resolution: calling the Go
// type mapper or canonical-family discovery here would recurse into itself.
// A demand is only a seed; the unchanged complete-family audit must still pass.
func sourceGenericViewDemandSeeds(ctx Ctx) []*symbol.ClassScope {
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

func sourceGenericViewDemandType(node *sitter.Node, ctx Ctx, source []byte) *symbol.ClassScope {
	switch resolvedSourceNodeType(node, ctx) {
	case "generic_type", "type_identifier", "scoped_type_identifier":
	default:
		return nil
	}
	parent := node.Parent()
	if parent != nil {
		if resolvedSourceNodeType(parent, ctx) == "scoped_type_identifier" {
			return nil // resolve the complete qualified spelling only
		}
		if resolvedSourceNodeType(parent, ctx) == "generic_type" && resolvedSourceNodeType(node, ctx) != "generic_type" {
			return nil // the base of Cell<String> is not a raw Cell occurrence
		}
	}
	javaType := node.Content(source)
	base, arguments := parseJavaTypeString(javaType)
	if visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || sourceGenericViewLocalTypeShadowed(node, base, source) {
		return nil
	}
	demanded := sourceGenericViewCastType(node)
	if resolvedSourceNodeType(node, ctx) == "generic_type" {
		// Cell<> is diamond inference, not a raw source view. Cell<String>
		// alone needs no additional migration; casts and wildcard views do.
		for _, argument := range arguments {
			demanded = demanded || strings.HasPrefix(strings.TrimSpace(argument), "?")
		}
	} else {
		demanded = true
	}
	if !demanded {
		return nil
	}
	seed := resolveClassScopeByQualifiedName(ctx, base)
	if seed == nil || seed.Class == nil || seed.Class.DeclarationNode == nil || len(seed.TypeParameters) == 0 {
		return nil
	}
	return seed
}

func sourceGenericViewCastType(node *sitter.Node) bool {
	for parent := node.Parent(); parent != nil; parent = node.Parent() {
		if parent.Type() == "array_type" || parent.Type() == "annotated_type" {
			node = parent
			continue
		}
		return parent.Type() == "cast_expression" && sameSourceNode(node, parent.ChildByFieldName("type"))
	}
	return false
}

// Match the declaration kinds registered as source ClassScopes. A nested record
// has its own binders just as a nested class does; visiting either in its outer
// declaration context would confuse a private type name with an imported one.
func sourceGenericViewTypeDeclaration(node *sitter.Node, contexts ...Ctx) bool {
	var ctx Ctx
	if len(contexts) > 0 {
		ctx = contexts[0]
	}
	if node == nil {
		return false
	}
	switch resolvedSourceNodeType(node, ctx) {
	case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration":
		return true
	}
	return false
}

// Local type declarations are not resolved source scopes until hoisting. Their
// name is in scope only in their declaring block, from the declaration onward.
// Fail closed for that name instead of treating it as an imported source class.
// Qualified p.Cell remains usable beside a local Cell; a local p also shadows
// the first segment of p.Cell, as Java's type-name namespace requires.
func sourceGenericViewLocalTypeShadowed(node *sitter.Node, base string, source []byte) bool {
	name := strings.SplitN(strings.TrimSpace(base), ".", 2)[0]
	declaredBefore := func(child *sitter.Node) bool {
		if child == nil || child.StartByte() >= node.StartByte() {
			return false
		}
		if sourceGenericViewTypeDeclaration(child) {
			declaredName := child.ChildByFieldName("name")
			return declaredName != nil && declaredName.Content(source) == name
		}
		return false
	}
	for scope := node.Parent(); scope != nil; scope = scope.Parent() {
		switch scope.Type() {
		case "block", "constructor_body", "switch_block_statement_group":
			for _, child := range nodeutil.NamedChildrenOf(scope) {
				if declaredBefore(child) {
					return true
				}
			}
		}
	}
	return false
}
