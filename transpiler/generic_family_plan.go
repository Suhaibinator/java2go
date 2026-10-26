package transpiler

import (
	"fmt"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// genericFamilyPlan inventories a connected source hierarchy before changing
// any physical layout. It deliberately does not select code generation yet:
// descriptor bridges, anonymous implementations and all member storage must
// consume the same successful plan before family lowering can be activated.
type genericFamilyPlan struct {
	storageInventory         *genericFamilyInventory
	pendingStorageComponents map[*symbol.ClassScope]struct{}
	members                  map[*symbol.ClassScope]struct{}
	anonymous                map[genericFamilySourceKey]genericFamilyAnonymous
	locals                   map[genericFamilySourceKey]genericFamilyAnonymous
	binders                  map[*symbol.TypeParamDeclaration]struct{}
	representations          map[*symbol.TypeParamDeclaration]genericFamilyBinderRepresentation
	constructorBodies        map[*symbol.Definition]*genericFamilyConstructorBodyCertificate
}

type genericFamilySourceKey struct {
	file       *symbol.FileScope
	start, end uint32
}

type genericFamilyAnonymous struct {
	owner          *symbol.ClassScope
	parent         *symbol.ClassScope
	node           *sitter.Node
	javaType       string
	carried        []symbol.TypeParam
	declarationCtx Ctx
	captures       []genericFamilyCaptureSnapshot
}

func genericFamilyKey(file *symbol.FileScope, node *sitter.Node) genericFamilySourceKey {
	return genericFamilySourceKey{file: file, start: node.StartByte(), end: node.EndByte()}
}

func planGenericFamily(seed *symbol.ClassScope, ctx Ctx) (*genericFamilyPlan, error) {
	return planGenericFamilyWithInventory(seed, ctx, nil)
}

// A discovery audits every seed independently, but its source declarations and
// hierarchy edges stay unchanged throughout those audits. Reuse only that
// structural inventory; eligibility and physical storage remain plan-owned.
type genericFamilyInventory struct {
	storageComponents       map[*symbol.ClassScope]*symbol.ClassScope
	storageComponentScans   int
	storageAuditRounds      int
	storageScopesAudited    int
	storageSourceEventScans int
	storageOwnership        *resolutionFileIndex
	storageOwnershipChecked bool
	scopes                  []*symbol.ClassScope
	localInterfaces         [][]*symbol.ClassScope
	parents                 map[*symbol.ClassScope][]*symbol.ClassScope
	sourceEvents            []genericFamilySourceEvent
	eventsLoaded            bool
}

func newGenericFamilyInventory(ctx Ctx) *genericFamilyInventory {
	scopes := allSourceClassScopes()
	inventory := &genericFamilyInventory{
		scopes:          scopes,
		localInterfaces: genericFamilyLocalInterfaceEdges(scopes, ctx),
		parents:         make(map[*symbol.ClassScope][]*symbol.ClassScope, len(scopes)),
	}
	for _, scope := range scopes {
		dependencies := genericFamilyParents(scope, ctx)
		// An inner allocation stores its enclosing instance. Their physical
		// layouts must migrate together, independently of source inheritance.
		if enclosing := genericFamilyEnclosingLayoutDependency(scope); enclosing != nil {
			present := false
			for _, dependency := range dependencies {
				present = present || dependency == enclosing
			}
			if !present {
				dependencies = append(dependencies, enclosing)
			}
		}
		inventory.parents[scope] = dependencies
	}
	return inventory
}

// An enclosing layout dependency is proven by carried declarations, never by
// source spelling. Static members and older synthetic scopes with no hidden
// declaration view keep their existing independent layout and audit behavior.
func genericFamilyEnclosingLayoutDependency(scope *symbol.ClassScope) *symbol.ClassScope {
	if scope == nil || !scope.IsInner || scope.Enclosing == nil || len(scope.Enclosing.TypeParameters) == 0 {
		return nil
	}
	hidden := len(scope.TypeParameters) - len(scope.OwnTypeParameters())
	if hidden < len(scope.Enclosing.TypeParameters) {
		return nil
	}
	carried := make(map[*symbol.TypeParamDeclaration]struct{}, hidden)
	for _, parameter := range scope.TypeParameters[:hidden] {
		if parameter.Declaration != nil {
			carried[parameter.Declaration] = struct{}{}
		}
	}
	for _, parameter := range scope.Enclosing.TypeParameters {
		if parameter.Declaration == nil {
			return nil
		}
		if _, present := carried[parameter.Declaration]; !present {
			return nil
		}
	}
	return scope.Enclosing
}

// A source event contains declaration structure only. In particular it never
// retains a synthesized local scope, a binder representation, or an admission
// result. Events preserve source traversal order and lexical method ownership.
type genericFamilySourceEvent struct {
	file       *symbol.FileScope
	owner      *symbol.ClassScope
	parent     *symbol.ClassScope
	node, body *sitter.Node
	lexical    Ctx
	javaType   string
	local      bool
	superclass bool
	err        error
}

func (inventory *genericFamilyInventory) genericFamilySourceEvents(ctx Ctx) []genericFamilySourceEvent {
	if inventory.eventsLoaded {
		return inventory.sourceEvents
	}
	inventory.eventsLoaded = true
	inventory.storageSourceEventScans++
	for _, owner := range inventory.scopes {
		if owner.Class == nil || owner.Class.DeclarationNode == nil {
			continue
		}
		file := findFileScopeForClassScope(owner, ctx)
		if file == nil {
			inventory.sourceEvents = append(inventory.sourceEvents, genericFamilySourceEvent{err: fmt.Errorf("missing source for generic family inventory")})
			break
		}
		var scan func(*sitter.Node, Ctx)
		scan = func(node *sitter.Node, lexical Ctx) {
			if node == nil {
				return
			}
			kind := node.Type()
			if node != owner.Class.DeclarationNode && (kind == "class_declaration" || kind == "interface_declaration" || kind == "enum_declaration") {
				parent := node.Parent()
				if parent != nil && parent.Type() != "class_body" && parent.Type() != "program" {
					for _, field := range []string{"interfaces", "superclass"} {
						if clause := node.ChildByFieldName(field); clause != nil {
							for _, typ := range collectTypeNodes(clause) {
								javaType := typ.Content(file.Source)
								base, _ := parseJavaTypeString(javaType)
								inventory.sourceEvents = append(inventory.sourceEvents, genericFamilySourceEvent{file: file, owner: owner, parent: resolveClassScopeByQualifiedName(lexical, base), node: node, body: node.ChildByFieldName("body"), lexical: lexical, javaType: javaType, local: true, superclass: field == "superclass"})
							}
						}
					}
				}
				return
			}
			if kind == "method_declaration" || kind == "constructor_declaration" {
				for _, method := range owner.Methods {
					if method.DeclarationNode != nil && method.DeclarationNode.StartByte() == node.StartByte() && method.DeclarationNode.EndByte() == node.EndByte() {
						lexical = lexical.Clone()
						lexical.localScope = method
						break
					}
				}
			}
			if kind == "object_creation_expression" {
				if typ := node.ChildByFieldName("type"); typ != nil {
					javaType := typ.Content(file.Source)
					base, _ := parseJavaTypeString(javaType)
					parent := resolveClassScopeByQualifiedName(lexical, base)
					for _, child := range nodeutil.NamedChildrenOf(node) {
						if child.Type() == "class_body" {
							inventory.sourceEvents = append(inventory.sourceEvents, genericFamilySourceEvent{file: file, owner: owner, parent: parent, node: node, body: child, lexical: lexical, javaType: javaType})
						}
					}
				}
			}
			for _, child := range nodeutil.NamedChildrenOf(node) {
				scan(child, classDeclarationChildCtx(node, child, lexical))
			}
		}
		scan(owner.Class.DeclarationNode, classScopeCtx(owner, ctx))
	}
	return inventory.sourceEvents
}

func planGenericFamilyWithInventory(seed *symbol.ClassScope, ctx Ctx, inventory *genericFamilyInventory) (*genericFamilyPlan, error) {
	if seed == nil || seed.Class == nil || len(seed.TypeParameters) == 0 {
		return nil, fmt.Errorf("generic family requires a source generic declaration")
	}
	if inventory == nil {
		inventory = newGenericFamilyInventory(ctx)
	}
	return planGenericFamilyWithStorageSeeds(seed, ctx, inventory, nil)
}

// Dependency rounds restart the unchanged complete audit. No provisional
// member, body certificate or anonymous scope escapes an unfinished round.
func planGenericFamilyWithStorageSeeds(seed *symbol.ClassScope, ctx Ctx, inventory *genericFamilyInventory, storageSeeds map[*symbol.ClassScope]struct{}) (*genericFamilyPlan, error) {
	if seed == nil || seed.Class == nil || len(seed.TypeParameters) == 0 {
		return nil, fmt.Errorf("generic family requires a source generic declaration")
	}
	plan := &genericFamilyPlan{
		members:         map[*symbol.ClassScope]struct{}{seed: {}},
		anonymous:       map[genericFamilySourceKey]genericFamilyAnonymous{},
		locals:          map[genericFamilySourceKey]genericFamilyAnonymous{},
		binders:         map[*symbol.TypeParamDeclaration]struct{}{},
		representations: map[*symbol.TypeParamDeclaration]genericFamilyBinderRepresentation{},
	}
	if inventory == nil {
		inventory = newGenericFamilyInventory(ctx)
	}
	inventory.storageAuditRounds++
	plan.storageInventory = inventory
	for dependency := range storageSeeds {
		plan.members[dependency] = struct{}{}
	}
	scopes := inventory.scopes
	localInterfaces := inventory.localInterfaces
	// Treat implemented source interfaces as hierarchy edges too. Discovering
	// another implementor later cannot change an already emitted method ABI.
	changed := true
	for changed {
		changed = false
		// A local implementation connects all of its source interfaces just as
		// a named implementation does. Select one ABI for the complete component
		// before inventorying or emitting any of its bridges.
		for _, parents := range localInterfaces {
			connected := false
			for _, parent := range parents {
				_, member := plan.members[parent]
				connected = connected || member
			}
			if !connected {
				continue
			}
			for _, parent := range parents {
				if _, member := plan.members[parent]; !member {
					plan.members[parent] = struct{}{}
					changed = true
				}
			}
		}
		for _, scope := range scopes {
			parents := inventory.parents[scope]
			for _, parent := range parents {
				_, childPresent := plan.members[scope]
				_, parentPresent := plan.members[parent]
				if !childPresent && !parentPresent {
					continue
				}
				if !childPresent {
					plan.members[scope] = struct{}{}
					changed = true
				}
				if !parentPresent {
					plan.members[parent] = struct{}{}
					changed = true
				}
			}
		}
	}
	for scope := range plan.members {
		if scope.IsEnum || scope.Class == nil || scope.Class.DeclarationNode == nil {
			return nil, fmt.Errorf("generic family contains an unsupported source declaration")
		}
		if err := plan.addBinderRepresentations(scope, ctx); err != nil {
			return nil, fmt.Errorf("generic family %s: %w", scope.Class.OriginalName, err)
		}
	}
	// Source traversal is shared by the discovery. Eligibility, synthesized
	// scopes, binder representations and storage audits remain fresh per plan.
	for _, event := range inventory.genericFamilySourceEvents(ctx) {
		if event.err != nil {
			return nil, event.err
		}
		if _, member := plan.members[event.parent]; !member {
			continue
		}
		if event.superclass {
			return nil, fmt.Errorf("local subclass requires a planned source scope")
		}
		if event.local {
			parameters := planLocalClassTypeParameters(event.node, nil, event.file.Source, event.lexical)
			local := synthLocalClassScope(event.node, event.body, event.node.ChildByFieldName("name").Content(event.file.Source), "", nil, nil, nil, parameters.carried, parameters.declared, anonymousClassMethods(event.body), localClassConstructors(event.body), nil, event.file.Source)
			local.Enclosing = event.owner
			localCtx := event.lexical.Clone()
			localCtx.currentClass = local
			localCtx.localScope = nil
			if err := plan.addBinderRepresentations(local, localCtx); err != nil {
				return nil, err
			}
			if !plan.typeSupported(event.javaType, localCtx, true, canonicalClassFamilyTypeOccurrence(event.javaType, event.node, event.file.Source)) {
				return nil, fmt.Errorf("local generic family specialization needs shared storage: %s", event.javaType)
			}
			if err := plan.auditTypeSyntax(event.node, event.file.Source, localCtx); err != nil {
				return nil, err
			}
			plan.locals[genericFamilyKey(event.file, event.node)] = genericFamilyAnonymous{owner: event.owner, parent: event.parent, node: event.node, javaType: event.javaType}
			continue
		}
		proof, captures, err := genericFamilyAnonymousAuditScope(event)
		if err != nil {
			return nil, err
		}
		anonymousCtx := classScopeCtx(proof, event.lexical)
		if err := plan.addBinderParameterRepresentations(proof.TypeParameters, event.lexical); err != nil {
			return nil, err
		}
		if proof.IsInner && proof.Enclosing != nil && len(proof.Enclosing.TypeParameters) != 0 {
			if _, present := plan.members[proof.Enclosing]; !present {
				return nil, fmt.Errorf("anonymous generic family enclosing instance requires a canonical layout")
			}
		}
		if !plan.typeSupported(event.javaType, event.lexical, true, canonicalClassFamilyTypeOccurrence(event.javaType, event.node, event.file.Source)) {
			return nil, fmt.Errorf("anonymous generic family specialization needs shared storage: %s", event.javaType)
		}
		for _, capture := range captures {
			occurrence := genericFamilyCaptureTypeOccurrence(capture, event)
			if !plan.typeSupported(capture.javaDef.OriginalType, event.lexical, true, occurrence) {
				return nil, fmt.Errorf("anonymous generic family capture requires unsupported physical storage: %s", capture.javaDef.OriginalType)
			}
		}
		captureProofs, err := genericFamilyCaptureSnapshots(captures, event.lexical)
		if err != nil {
			return nil, err
		}
		if err := plan.auditTypeSyntax(event.body, event.file.Source, anonymousCtx); err != nil {
			return nil, err
		}
		plan.anonymous[genericFamilyKey(event.file, event.node)] = genericFamilyAnonymous{owner: event.owner, parent: event.parent, node: event.node, javaType: event.javaType, carried: append([]symbol.TypeParam(nil), proof.TypeParameters...), declarationCtx: event.lexical, captures: captureProofs}
	}
	audited := map[*symbol.ClassScope]bool{}
	for len(audited) < len(plan.members) {
		for scope := range plan.members {
			if audited[scope] {
				continue
			}
			audited[scope] = true
			inventory.storageScopesAudited++
			declarationCtx := classScopeCtx(scope, ctx)
			for _, method := range scope.Methods {
				if method == nil || !method.Constructor {
					continue
				}
				for _, parameter := range scope.TypeParameters {
					if !constructorBodySupportsCallableErasure(scope, method, parameter.Declaration, ctx) && !certifyGenericFamilyEnumConstructorBody(plan, scope, method, parameter.Declaration, ctx) {
						return nil, fmt.Errorf("generic family constructor requires an erased body entry: %s", scope.Class.OriginalName)
					}
				}
			}
			if scope.IsInner && scope.Enclosing != nil && len(scope.Enclosing.TypeParameters) != 0 {
				if _, present := plan.members[scope.Enclosing]; !present {
					return nil, fmt.Errorf("generic family enclosing instance requires a canonical layout: %s", scope.Class.OriginalName)
				}
			}
			for _, parent := range append([]string{scope.Superclass}, scope.ImplementedInterfaces...) {
				if !plan.typeSupported(parent, classHeaderTypeCtx(scope, declarationCtx), true, canonicalClassFamilyHeaderTypeOccurrence(parent, scope.Class.DeclarationNode, declarationCtx)) {
					return nil, fmt.Errorf("generic family specialization needs shared storage: %s", parent)
				}
			}
			file := findFileScopeForClassScope(scope, ctx)
			if file == nil {
				return nil, fmt.Errorf("missing generic family declaration source")
			}
			if err := plan.auditTypeSyntax(scope.Class.DeclarationNode, file.Source, declarationCtx); err != nil {
				return nil, err
			}
		}
	}
	if len(plan.pendingStorageComponents) != 0 {
		next := make(map[*symbol.ClassScope]struct{}, len(storageSeeds)+len(plan.pendingStorageComponents))
		for dependency := range storageSeeds {
			next[dependency] = struct{}{}
		}
		for dependency := range plan.pendingStorageComponents {
			next[dependency] = struct{}{}
		}
		if len(next) <= len(storageSeeds) {
			return nil, fmt.Errorf("source storage component closure made no progress")
		}
		return planGenericFamilyWithStorageSeeds(seed, ctx, inventory, next)
	}
	plan.storageInventory = nil
	if err := certifyGenericFamilySourceBounds(plan, ctx); err != nil {
		return nil, err
	}
	return plan, nil
}

func genericFamilyParents(scope *symbol.ClassScope, ctx Ctx) []*symbol.ClassScope {
	if scope == nil {
		return nil
	}
	var result []*symbol.ClassScope
	declarationCtx := classHeaderTypeCtx(scope, ctx)
	for _, typ := range append([]string{scope.Superclass}, scope.ImplementedInterfaces...) {
		base, _ := parseJavaTypeString(typ)
		if parent := resolveClassScopeByQualifiedName(declarationCtx, base); parent != nil {
			result = append(result, parent)
		}
	}
	return result
}

// typeSupported checks source types, retaining declaration identity for class
// versus method binders. A runtime List<T>/Map<K,V> is not a source alias: Go
// pointer reinterpretation would break writes and Java's delayed checkcasts.
func (plan *genericFamilyPlan) typeSupported(typ string, ctx Ctx, specialization bool, occurrences ...*sitter.Node) bool {
	typ = strings.TrimSpace(typ)
	if typ == "" {
		return true
	}
	for _, prefix := range []string{"? extends ", "? super "} {
		typ = strings.TrimPrefix(typ, prefix)
	}
	component, rank := javaArrayTypeParts(typ)
	base, args := parseJavaTypeString(component)
	// Enum has one nominal erased object representation regardless of E.
	if rank == 0 && isBuiltinEnum(component, ctx) {
		return true
	}
	if genericFamilyCanonicalComparableHeader(typ, ctx, occurrences...) {
		return true
	}
	if canonicalClassArgumentIndependent(typ, ctx, occurrences...) {
		return true
	}
	if genericFamilyCollectionArgumentIndependent(typ, ctx, occurrences...) {
		return true
	}
	declaration := visibleTypeParameterDeclarationForJavaType(base, ctx)
	_, owned := plan.binders[declaration]
	if len(args) == 0 {
		return !owned || rank == 0 || plan.sourceBoundReferenceArray(declaration, rank)
	}
	containsOwned := false
	for binder := range plan.binders {
		if visibleTypeParameterDeclarationForJavaType(binder.SourceName, ctx) == binder && javaTypeContainsParameter(typ, binder.SourceName) {
			containsOwned = true
			break
		}
	}
	if !specialization && !containsOwned {
		return true
	}
	if mapReferenceArgumentIndependent(typ, ctx) || mapEntryArgumentIndependent(typ, ctx) {
		return true
	}
	target := resolveClassScopeByQualifiedName(ctx, base)
	_, member := plan.members[target]
	if !member {
		if !containsOwned {
			return false
		}
		if !genericFamilyStorageLeaf(target, ctx) {
			if !plan.queueSourceComponentStorage(target, typ, ctx, occurrences...) {
				return false
			}
			for _, argument := range args {
				if !plan.typeSupported(argument, ctx, true, occurrences...) {
					return false
				}
			}
			return true
		}
		// Nested source storage must activate with its owning family, including
		// transitive and cyclic dependencies. Eligibility alone is insufficient:
		// independent Go instantiations would split the shared Java allocation.
		plan.members[target] = struct{}{}
		if err := plan.addBinderRepresentations(target, ctx); err != nil {
			return false
		}
	}
	if rank != 0 && containsOwned {
		return false
	}
	for _, argument := range args {
		if !plan.typeSupported(argument, ctx, true, occurrences...) {
			return false
		}
	}
	return true
}

func (plan *genericFamilyPlan) auditTypeSyntax(root *sitter.Node, source []byte, ctx Ctx) error {
	var walk func(*sitter.Node, Ctx) error
	walk = func(node *sitter.Node, lexical Ctx) error {
		if node == nil {
			return nil
		}
		if node != root && (node.Type() == "class_declaration" || node.Type() == "interface_declaration" || node.Type() == "enum_declaration") {
			return nil
		}
		if node.Type() == "method_declaration" || node.Type() == "constructor_declaration" {
			found := false
			if lexical.currentClass != nil {
				for _, method := range lexical.currentClass.Methods {
					if method.DeclarationNode != nil && method.DeclarationNode.StartByte() == node.StartByte() && method.DeclarationNode.EndByte() == node.EndByte() {
						lexical = lexical.Clone()
						lexical.localScope = method
						found = true
						break
					}
				}
			}
			if !found && node.Type() == "method_declaration" {
				// Anonymous methods are not yet in the global symbol table. Reconstruct
				// their own binders before checking names; a method <T> must shadow the
				// enclosing class T even during this pre-emission inventory.
				method := scopeForAnonymousMethod(node, synthAnonClassMethodDefinition(node, source, false), source)
				lexical = lexical.Clone()
				lexical.localScope = method
			}
		}
		switch node.Type() {
		case "generic_type", "array_type", "type_identifier":
			if !plan.typeSupported(node.Content(source), lexical, false, node) {
				return fmt.Errorf("generic family type requires unsupported physical storage: %s", node.Content(source))
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			if err := walk(child, classDeclarationChildCtx(node, child, lexical)); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(root, ctx)
}

// Storage dependencies are source leaves; hierarchy edges are planned by the
// main family closure. Their complete bodies and constructors are audited by
// the same fixed-point worklist before any layout is activated.
func genericFamilyStorageLeaf(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil || scope.IsInterface || scope.IsAbstract || scope.IsEnum || scope.IsInner || len(scope.TypeParameters) == 0 || len(scope.ImplementedInterfaces) != 0 {
		return false
	}
	if scope.Superclass != "" && stripJavaQualifier(scope.Superclass) != "Object" {
		return false
	}
	for _, parameter := range scope.TypeParameters {
		if parameter.Declaration == nil || len(parameter.Bounds) > 1 || stripJavaQualifier(rawTypeParameterErasure(parameter, scope.TypeParameters)) != "Object" {
			return false
		}
	}
	for _, other := range allSourceClassScopes() {
		if other != scope && classScopeDescendsFrom(other, scope, ctx) {
			return false
		}
	}
	return !classHasUnmodeledCallableSubclass(scope, ctx)
}

// Local classes are absent from the named symbol graph. Their implemented
// interface lists still contribute hierarchy edges, including interfaces that
// have no independent universal-method demand of their own.
func genericFamilyLocalInterfaceEdges(scopes []*symbol.ClassScope, ctx Ctx) [][]*symbol.ClassScope {
	var edges [][]*symbol.ClassScope
	namedDeclarations := make(map[uintptr]bool, len(scopes))
	for _, scope := range scopes {
		if scope.Class != nil && scope.Class.DeclarationNode != nil {
			namedDeclarations[scope.Class.DeclarationNode.ID()] = true
		}
	}
	for _, owner := range scopes {
		if owner.Class == nil || owner.Class.DeclarationNode == nil {
			continue
		}
		file := findFileScopeForClassScope(owner, ctx)
		if file == nil {
			continue
		}
		declarationCtx := classScopeCtx(owner, ctx)
		var walk func(*sitter.Node)
		walk = func(node *sitter.Node) {
			if node == nil {
				return
			}
			if !node.Equal(owner.Class.DeclarationNode) && namedDeclarations[node.ID()] {
				// This declaration is scanned separately in its own lexical
				// owner context, including any interfaces shadowing outer names.
				return
			}
			if node != owner.Class.DeclarationNode && node.Type() == "class_declaration" {
				if parent := node.Parent(); parent != nil && parent.Type() != "class_body" && parent.Type() != "program" {
					var interfaces []*symbol.ClassScope
					if clause := node.ChildByFieldName("interfaces"); clause != nil {
						for _, typ := range collectTypeNodes(clause) {
							base, _ := parseJavaTypeString(typ.Content(file.Source))
							if resolved := resolveClassScopeByQualifiedName(declarationCtx, base); resolved != nil {
								interfaces = append(interfaces, resolved)
							}
						}
					}
					if len(interfaces) > 1 {
						edges = append(edges, interfaces)
					}
				}
			}
			for _, child := range nodeutil.NamedChildrenOf(node) {
				walk(child)
			}
		}
		walk(owner.Class.DeclarationNode)
	}
	return edges
}
