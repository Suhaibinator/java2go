package transpiler

import (
	"bytes"
	"os"
	"sort"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// One invocation owns copied syntax routes, never resolved seeds or declarations.
// File/root/node identities are certified by walking the actual owned parse tree.
// A route records only the root child and method syntax needed to reconstruct the
// original walk's lexical context from current symbols at query time.
type sourceGenericDemandSyntax struct {
	invocation *syntaxInvocation
	roots      map[*sitter.Node]sourceGenericDemandRoot
}
type sourceGenericDemandRoot struct {
	file   *symbol.FileScope
	routes []sourceGenericDemandRoute
}
type sourceGenericDemandRoute struct {
	child, method *sitter.Node
	nodes         []*sitter.Node
}

func sourceGenericViewDemandSeedsFromSyntax(ctx Ctx) ([]*symbol.ClassScope, bool) {
	invocation := ctx.syntaxInvocation
	if !sourceGenericDemandOwnedCompilationUnits(invocation) {
		return nil, false
	}
	proof, known := sourceGenericDemandOwnershipContext()
	if !known {
		return nil, false
	}
	owners := allSourceClassScopes()
	inventory := invocation.demandSyntax
	if inventory == nil {
		inventory = buildSourceGenericDemandSyntax(invocation, owners, proof)
		if inventory == nil {
			return nil, false
		}
		invocation.demandSyntax = inventory
	}
	if !inventory.matches(invocation, owners, proof) {
		return nil, false
	}
	names := sourceGenericDemandSourceNames(owners, ctx, proof.sourceOwnership)
	seen := map[*symbol.ClassScope]bool{}
	var seeds []*symbol.ClassScope
	for _, owner := range owners {
		if owner.Class == nil || owner.Class.DeclarationNode == nil {
			continue
		}
		root := owner.Class.DeclarationNode
		entry := inventory.roots[root]
		if len(entry.routes) == 0 {
			continue
		}
		base := classScopeCtx(owner, ctx)
		for _, route := range entry.routes {
			lexical := base
			if route.child != nil {
				lexical = classDeclarationChildCtx(root, route.child, lexical)
			}
			if route.method != nil {
				lexical = lexical.Clone()
				lexical.localScope = nil
				for _, method := range owner.Methods {
					if method.DeclarationNode != nil && method.DeclarationNode.Equal(route.method) {
						lexical.localScope = method
						break
					}
				}
			}
			for _, node := range route.nodes {
				if seed := sourceGenericViewDemandTypeFromNames(node, lexical, entry.file.Source, names); seed != nil && !seen[seed] {
					seen[seed] = true
					seeds = append(seeds, seed)
				}
			}
		}
	}
	sort.Slice(seeds, func(i, j int) bool { return qualifiedSourceClassName(seeds[i]) < qualifiedSourceClassName(seeds[j]) })
	return seeds, true
}

// Validate the complete query before any semantic replay. Source-byte equality
// catches in-place edits; a foreign root, graph or retired capability is unknown.
func sourceGenericDemandOwnedFiles(invocation *syntaxInvocation) bool {
	if invocation == nil || !invocation.active || !invocation.indexable || !invocation.session.Active() || invocation.graph != symbol.GlobalScope || !standaloneSyntaxConfigurationOK(os.Stdout) {
		return false
	}
	seen := make(map[*symbol.FileScope]bool, len(invocation.files))
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			entry := invocation.identifiers[file]
			if seen[file] || entry == nil || entry.anchor != file.ParsedRoot() || entry.anchor.HasChanges() || !bytes.Equal(entry.source, file.Source) {
				return false
			}
			seen[file] = true
		}
	}
	return len(seen) == len(invocation.files)
}

// A closed batch may also own inert package compilation units that deliberately
// have no FileScope registration. Prove each omitted unit from its current
// owned parse anchor; the legacy exact-membership predicate stays unchanged.
func sourceGenericDemandOwnedCompilationUnits(invocation *syntaxInvocation) bool {
	if invocation == nil || !invocation.active || !invocation.indexable || !invocation.session.Active() || invocation.graph != symbol.GlobalScope || !standaloneSyntaxConfigurationOK(os.Stdout) {
		return false
	}
	seen := make(map[*symbol.FileScope]bool, len(invocation.files))
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			entry := invocation.identifiers[file]
			if _, owned := invocation.files[file]; !owned {
				return false
			}
			if seen[file] || entry == nil || entry.anchor == nil || entry.anchor != file.ParsedRoot() || entry.anchor.HasChanges() || !bytes.Equal(entry.source, file.Source) {
				return false
			}
			seen[file] = true
		}
	}
	for file, owned := range invocation.files {
		entry := invocation.identifiers[file]
		if file == nil || owned.Symbols != file || entry == nil || entry.anchor == nil || owned.Ast == nil || entry.anchor != file.ParsedRoot() || !owned.Ast.Equal(entry.anchor) || entry.anchor.HasChanges() || !bytes.Equal(entry.source, file.Source) || !bytes.Equal(owned.Source, entry.source) {
			return false
		}
		if seen[file] {
			continue
		}
		if file.BaseClass != nil || len(file.TopLevelClasses) != 0 {
			return false
		}
		// Classify the current certified root, never a filename or a missing count.
		current := owned
		current.Ast, current.Source = entry.anchor, file.Source
		unit, err := classifyCompilationUnit(current)
		if err != nil || !unit.packageMetadata || unit.packageName == "" || unit.packageName != file.Package || !sourceGenericDemandInertCompilationUnit(entry.anchor) {
			return false
		}
	}
	return true
}

// Only the syntax forms needed by inert package statements, marker package
// annotations and imports are known here. Class literals, type demands,
// declarations and every unknown form fall back to ordinary discovery.
func sourceGenericDemandInertCompilationUnit(root *sitter.Node) bool {
	var visit func(*sitter.Node) bool
	visit = func(node *sitter.Node) bool {
		if node == nil || node.IsMissing() || node.HasError() || node.HasChanges() {
			return false
		}
		switch node.Type() {
		case "program", "package_declaration", "import_declaration", "marker_annotation", "identifier", "scoped_identifier", "line_comment", "block_comment":
		default:
			return false
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			if !visit(node.NamedChild(i)) {
				return false
			}
		}
		return true
	}
	return root != nil && root.Type() == "program" && visit(root)
}

// Ownership must come from current graph topology, independently of caller
// inventories. The proof context is temporary and supplies no lexical state.
func sourceGenericDemandOwnershipContext() (Ctx, bool) {
	ownership := newResolutionFileIndex()
	if active := activeResolutionFiles; active != nil && active.graph == symbol.GlobalScope {
		if len(active.files) != len(ownership.files) {
			return Ctx{}, false
		}
		for owner, file := range ownership.files {
			if active.files[owner] != file {
				return Ctx{}, false
			}
		}
	}
	return Ctx{sourceOwnership: ownership}, true
}

func (inventory *sourceGenericDemandSyntax) matches(invocation *syntaxInvocation, owners []*symbol.ClassScope, ctx Ctx) bool {
	if inventory.invocation != invocation {
		return false
	}
	seen := make(map[*sitter.Node]bool, len(inventory.roots))
	for _, owner := range owners {
		if owner.Class == nil || owner.Class.DeclarationNode == nil {
			continue
		}
		root := owner.Class.DeclarationNode
		entry, known := inventory.roots[root]
		if !known || seen[root] || findFileScopeForClassScope(owner, ctx) != entry.file {
			return false
		}
		seen[root] = true
	}
	return len(seen) == len(inventory.roots)
}

func buildSourceGenericDemandSyntax(invocation *syntaxInvocation, owners []*symbol.ClassScope, ctx Ctx) *sourceGenericDemandSyntax {
	inventory := &sourceGenericDemandSyntax{invocation: invocation, roots: make(map[*sitter.Node]sourceGenericDemandRoot)}
	targets := make(map[*sitter.Node]*symbol.FileScope)
	for _, owner := range owners {
		if owner.Class == nil || owner.Class.DeclarationNode == nil {
			continue
		}
		file := findFileScopeForClassScope(owner, ctx)
		if _, owned := invocation.files[file]; !owned {
			return nil
		}
		root := owner.Class.DeclarationNode
		if _, duplicate := targets[root]; duplicate {
			return nil
		}
		targets[root] = file
	}
	// Do not certify roots by names or byte ranges. Only actual nodes reached from
	// each attested parse anchor can become inventory roots.
	for file := range invocation.files {
		var visit func(*sitter.Node) bool
		visit = func(node *sitter.Node) bool {
			if node == nil {
				return true
			}
			if targetFile, known := targets[node]; known {
				if targetFile != file || !sourceGenericViewTypeDeclaration(node, ctx) {
					return false
				}
				inventory.roots[node] = sourceGenericDemandRoot{file: file, routes: sourceGenericDemandRoutes(node, ctx)}
			}
			for _, child := range resolvedSourceNamedChildren(node, ctx) {
				if !visit(child) {
					return false
				}
			}
			return true
		}
		if !visit(file.ParsedRoot()) {
			return nil
		}
	}
	if len(inventory.roots) != len(targets) {
		return nil
	}
	return inventory
}

func sourceGenericDemandRoutes(root *sitter.Node, ctx Ctx) []sourceGenericDemandRoute {
	var routes []sourceGenericDemandRoute
	var walk func(*sitter.Node, *sitter.Node, *sitter.Node)
	walk = func(node, rootChild, method *sitter.Node) {
		if node == nil {
			return
		}
		kind := resolvedSourceNodeType(node, ctx)
		if sourceGenericViewTypeDeclaration(node, ctx) && node != root {
			return
		}
		if kind == "class_body" {
			parent := node.Parent()
			if parent != nil && (resolvedSourceNodeType(parent, ctx) == "object_creation_expression" || resolvedSourceNodeType(parent, ctx) == "enum_constant") {
				return
			}
		}
		if kind == "method_declaration" || kind == "constructor_declaration" {
			method = node
		}
		switch kind {
		case "generic_type", "type_identifier", "scoped_type_identifier":
			if len(routes) == 0 || routes[len(routes)-1].child != rootChild || routes[len(routes)-1].method != method {
				routes = append(routes, sourceGenericDemandRoute{child: rootChild, method: method})
			}
			last := &routes[len(routes)-1]
			last.nodes = append(last.nodes, node)
		}
		for _, child := range resolvedSourceNamedChildren(node, ctx) {
			nextChild := rootChild
			if node == root {
				nextChild = child
			}
			walk(child, nextChild, method)
		}
	}
	walk(root, nil, nil)
	return routes
}
