package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// An occurrence can name either an interface or one of its concrete source
// implementations, or a generic source class component. Queue an actual source
// declaration anchor; the next fresh audit must certify that entire connected component before ABI use.
func (plan *genericFamilyPlan) queueSourceComponentStorage(target *symbol.ClassScope, javaType string, ctx Ctx, occurrences ...*sitter.Node) bool {
	if plan == nil || plan.storageInventory == nil || target == nil || target.Class == nil || target.Class.DeclarationNode == nil || len(target.TypeParameters) == 0 || ctx.currentFile == nil || len(occurrences) != 1 || occurrences[0] == nil {
		return false
	}
	kind := target.Class.DeclarationNode.Type()
	if kind != "class_declaration" && kind != "interface_declaration" {
		return false
	}
	component, rank := javaArrayTypeParts(javaType)
	base, args := parseJavaTypeString(component)
	if rank != 0 || len(args) != len(target.TypeParameters) {
		return false
	}
	inventory := plan.storageInventory
	registered := false
	for _, scope := range inventory.scopes {
		if scope == target {
			registered = true
			break
		}
	}
	if !registered {
		return false
	}
	seed := inventory.sourceStorageComponentAnchor(target)
	if seed == nil {
		return false
	}
	occurrence := occurrences[0]
	if occurrence.EndByte() > uint32(len(ctx.currentFile.Source)) || occurrence.StartByte() >= occurrence.EndByte() {
		return false
	}
	root := occurrence
	for root.Parent() != nil {
		root = root.Parent()
	}
	anchor := ctx.currentFile.ParsedRoot()
	if anchor == nil || !root.Equal(anchor) {
		return false
	}
	// Establish independent current-graph ownership before any binder or name
	// resolver can touch an incoming ownership/declaration-facts holder.
	if !inventory.storageOwnershipChecked {
		inventory.storageOwnershipChecked = true
		if proof, known := sourceGenericDemandOwnershipContext(); known && genericFamilyOwnershipUnique(proof.sourceOwnership) {
			inventory.storageOwnership = proof.sourceOwnership
		}
	}
	ownership := inventory.storageOwnership
	if ownership == nil || ownership.graph != symbol.GlobalScope {
		return false
	}
	occurrenceFileKnown := false
	for _, file := range ownership.files {
		if file == ctx.currentFile {
			occurrenceFileKnown = true
			break
		}
	}
	if !occurrenceFileKnown {
		return false
	}
	if !sourceStorageDeclarationOwned(target, ownership) || !sourceStorageDeclarationOwned(seed, ownership) {
		return false
	}
	lookup := ctx.Clone()
	lookup.sourceOwnership = ownership
	lookup.syntaxInvocation = nil
	lookup.localClasses = nil
	if visibleTypeParameterDeclarationForJavaType(strings.SplitN(base, ".", 2)[0], lookup) != nil {
		return false
	}
	// Recursive argument checks may retain an outer type node. Recover only the
	// exact nested TYPE occurrence from the original, registered source file.
	var use *sitter.Node
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "generic_type" && node.Content(lookup.currentFile.Source) == javaType {
			if use == nil {
				use = node
			}
			return
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			visit(child)
		}
	}
	visit(occurrence)
	if use == nil || sourceGenericViewLocalTypeShadowed(use, base, lookup.currentFile.Source) {
		return false
	}
	if resolveClassScopeByQualifiedName(lookup, base) != target {
		return false
	}
	if plan.pendingStorageComponents == nil {
		plan.pendingStorageComponents = make(map[*symbol.ClassScope]struct{})
	}
	plan.pendingStorageComponents[seed] = struct{}{}
	return true
}

func sourceStorageDeclarationOwned(scope *symbol.ClassScope, ownership *resolutionFileIndex) bool {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil {
		return false
	}
	file := ownership.files[scope]
	if file == nil || file.ParsedRoot() == nil {
		return false
	}
	node := scope.Class.DeclarationNode
	name := node.ChildByFieldName("name")
	if node.EndByte() > uint32(len(file.Source)) || name == nil || name.Content(file.Source) != scope.Class.OriginalName {
		return false
	}
	root := node
	for root.Parent() != nil {
		root = root.Parent()
	}
	return root.Equal(file.ParsedRoot())
}

// Cache only immutable structural edges already inventoried by the planner.
// No admission, synthesized scope, binder representation or partial plan is
// stored here. A stable full audit still decides whether the component works.
func (inventory *genericFamilyInventory) sourceStorageComponentAnchor(target *symbol.ClassScope) *symbol.ClassScope {
	if inventory.storageComponents != nil {
		return inventory.storageComponents[target]
	}
	inventory.storageComponentScans++
	inventory.storageComponents = make(map[*symbol.ClassScope]*symbol.ClassScope, len(inventory.scopes))
	registered := make(map[*symbol.ClassScope]bool, len(inventory.scopes))
	for _, scope := range inventory.scopes {
		registered[scope] = true
	}
	edges := make(map[*symbol.ClassScope][]*symbol.ClassScope, len(inventory.scopes))
	connect := func(a, b *symbol.ClassScope) {
		if a == nil || b == nil || !registered[a] || !registered[b] {
			return
		}
		edges[a] = append(edges[a], b)
		edges[b] = append(edges[b], a)
	}
	for child, parents := range inventory.parents {
		for _, parent := range parents {
			connect(child, parent)
		}
	}
	for _, parents := range inventory.localInterfaces {
		if len(parents) > 1 {
			for _, parent := range parents[1:] {
				connect(parents[0], parent)
			}
		}
	}
	seen := make(map[*symbol.ClassScope]bool, len(inventory.scopes))
	for _, first := range inventory.scopes {
		if seen[first] {
			continue
		}
		seen[first] = true
		queue := []*symbol.ClassScope{first}
		var seed *symbol.ClassScope
		var sourceClass *symbol.ClassScope
		for index := 0; index < len(queue); index++ {
			scope := queue[index]
			if seed == nil && scope.IsInterface && len(scope.TypeParameters) > 0 && scope.Class != nil && scope.Class.DeclarationNode != nil && scope.Class.DeclarationNode.Type() == "interface_declaration" {
				seed = scope
			}
			if sourceClass == nil && !scope.IsInterface && scope.Class != nil && len(scope.TypeParameters) > 0 && scope.Class.DeclarationNode != nil && scope.Class.DeclarationNode.Type() == "class_declaration" {
				sourceClass = scope
			}
			for _, next := range edges[scope] {
				if !seen[next] {
					seen[next] = true
					queue = append(queue, next)
				}
			}
		}
		if seed == nil {
			seed = sourceClass
		}
		for _, scope := range queue {
			inventory.storageComponents[scope] = seed
		}
	}
	return inventory.storageComponents[target]
}
