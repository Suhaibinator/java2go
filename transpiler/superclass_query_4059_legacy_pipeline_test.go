package transpiler

import (
	"go/ast"
	"sort"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// The allocation oracle must retain its pre233 transitive discovery path. The
// current compiler's ownership/name inventories are the optimization measured
// by this test and cannot serve as the old side of the same measurement.
// These are public232 source functions with capsule-local names. Semantic type
// conversion, source definitions, imports, AST objects, and entry shapes remain
// the real compiler objects; no scan/counter work is added for budget padding.
// Frozen public232 function: sourceInterfaceVirtualSignature; only capsule call names change.
func superclassQuery4059LegacySourceInterfaceVirtualSignature(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) *ast.FuncType {
	params := &ast.FieldList{}
	for index, parameter := range method.Parameters {
		if parameter == nil {
			return nil
		}
		javaType := qualifyJavaTypeInDeclaringContext(definitionParameterJavaSignatureType(method, index), owner)
		parameterType := abstractClassToInterface(javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx), javaType, ctx)
		parameterType = rawUnboundReceiverParameterType(owner, method, index, javaType, parameterType, ctx)
		params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(parameter.Name)}, Type: parameterType})
	}
	var results *ast.FieldList
	if strings.TrimSpace(method.OriginalType) != "" && strings.TrimSpace(method.OriginalType) != "void" {
		javaType := qualifyJavaTypeInDeclaringContext(method.OriginalType, owner)
		results = &ast.FieldList{List: []*ast.Field{{Type: abstractClassToInterface(javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx), javaType, ctx)}}}
	}
	public := &ast.Field{Type: &ast.FuncType{Params: params, Results: results}}
	return superclassQuery4059LegacyExecutionMethodField(public, method, owner, ctx).Type.(*ast.FuncType)
}

// Frozen public232 function: executionMethodField; only capsule call names change.
func superclassQuery4059LegacyExecutionMethodField(public *ast.Field, def *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) *ast.Field {
	if public == nil || def == nil {
		return nil
	}
	functionType, ok := public.Type.(*ast.FuncType)
	if !ok || functionType == nil {
		return nil
	}
	params := cloneFieldList(functionType.Params)
	if params == nil {
		params = &ast.FieldList{}
	}
	var reservedNames []string
	if owner != nil {
		reservedNames = append(reservedNames, owner.GoTypeParameterNames()...)
	}
	reservedNames = append(reservedNames, def.GoTypeParameterNames()...)
	executionName := executionNameForParams(params, reservedNames...)
	params.List = append([]*ast.Field{executionParameterField(executionName, ctx)}, params.List...)
	return &ast.Field{
		Names: []*ast.Ident{{Name: superclassQuery4059LegacyExecutionImplementationName(def, owner, ctx)}},
		Type: &ast.FuncType{
			Params:  params,
			Results: cloneFieldList(functionType.Results),
		},
	}
}

// Frozen public232 function: executionImplementationName; only capsule call names change.
func superclassQuery4059LegacyExecutionImplementationName(def *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) string {
	if def == nil {
		return ""
	}
	if name := sourceCollectionAddImplementationName(def, owner); name != "" {
		return name
	}
	if selection, bridged := superclassQuery4059LegacyDirectOwnerSpecializedOverrideBridgeForMethod(owner, def, superclassQuery4059LegacyClassScopeCtx(owner, ctx)); bridged {
		return directOwnerOverrideBridgeExactExecutionName(selection.bridge)
	}
	if name := mapEntrySourceImplementationName(def, owner, ctx); name != "" {
		return name
	}
	if name := iterationSourceImplementationName(def, owner, ctx); name != "" {
		return name
	}
	return collisionSafeExecutionIdentifier(def.Name+executionMethodSuffix, owner)
}

// Frozen public232 function: directOwnerSpecializedOverrideBridgeForMethod; only capsule call names change.
func superclassQuery4059LegacyDirectOwnerSpecializedOverrideBridgeForMethod(
	owner *symbol.ClassScope,
	method *symbol.Definition,
	ctx Ctx,
) (directOwnerSpecializedOverrideBridgeSelection, bool) {
	if owner == nil || method == nil || method.Constructor || method.IsStatic || method.IsPrivate {
		return directOwnerSpecializedOverrideBridgeSelection{}, false
	}
	if selection, ok := superclassQuery4059LegacyGenericFamilySpecializedMethod(owner, method, ctx); ok {
		return selection, true
	}
	ctx = superclassQuery4059LegacyClassScopeCtx(owner, ctx)
	var selected directOwnerSpecializedOverrideBridgeSelection
	descriptorKey := ""
	seen := make(map[*symbol.ClassScope]struct{})
	for ancestor := resolveSuperclassScopeInDeclaringContext(ctx, owner); ancestor != nil; ancestor = resolveSuperclassScopeInDeclaringContext(ctx, ancestor) {
		if _, duplicate := seen[ancestor]; duplicate {
			return directOwnerSpecializedOverrideBridgeSelection{}, false
		}
		seen[ancestor] = struct{}{}
		for _, candidate := range ancestor.Methods {
			if candidate == nil || candidate.Constructor || candidate.IsStatic || candidate.IsPrivate ||
				candidate.OriginalName != method.OriginalName || len(candidate.Parameters) != len(method.Parameters) {
				continue
			}
			family, ok := planDirectOwnerCallableOverrideBridgeFamily(ancestor, candidate, ctx)
			if !ok || family == nil {
				continue
			}
			for _, bridge := range family.overrides {
				if bridge.owner != owner || bridge.method != method {
					continue
				}
				key := overrideBridgeFamilyDescriptorKey(family, ctx)
				if descriptorKey == "" {
					descriptorKey = key
					selected = directOwnerSpecializedOverrideBridgeSelection{family: family, bridge: bridge}
					continue
				}
				if key != descriptorKey {
					return directOwnerSpecializedOverrideBridgeSelection{}, false
				}
			}
		}
	}
	if selected.family == nil {
		return specializedAncestorCovariantBridge(owner, method, ctx)
	}
	return selected, selected.family != nil
}

// Frozen public232 function: genericFamilySpecializedMethod; only capsule call names change.
func superclassQuery4059LegacyGenericFamilySpecializedMethod(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) (directOwnerSpecializedOverrideBridgeSelection, bool) {
	empty := directOwnerSpecializedOverrideBridgeSelection{}
	layout := superclassQuery4059LegacyCanonicalGenericFamily(owner, ctx)
	if layout == nil {
		return empty, false
	}
	var selected directOwnerSpecializedOverrideBridgeSelection
	for ancestor := range layout.members {
		if ancestor == owner || len(ancestor.TypeParameters) == 0 {
			continue
		}
		args := mapClassTypeArgumentStringsToAncestor(owner, owner.GoTypeParameterNames(), ancestor, ctx)
		if len(args) != len(ancestor.TypeParameters) {
			continue
		}
		for _, candidate := range ancestor.Methods {
			if candidate == nil || candidate.OriginalName != method.OriginalName {
				continue
			}
			family, ok := superclassQuery4059LegacyGenericFamilyMethodPlan(ancestor, candidate, ctx)
			if !ok {
				continue
			}
			parameters, result := mappedOverrideSourceSignature(ancestor, candidate, args)
			matches := directDeclaredOverrides(ancestor, owner, candidate, parameters, result, ctx)
			matched := false
			for _, match := range matches {
				matched = matched || match == method
			}
			if !matched {
				continue
			}
			bridge, required, ok := planDirectOwnerSpecializedOverride(ancestor, candidate, owner, method, args, parameters, result, family.erasedParameters, family.erasedResult, ctx)
			if !ok || !required {
				continue
			}
			if selected.family != nil && overrideBridgeFamilyDescriptorKey(selected.family, ctx) != overrideBridgeFamilyDescriptorKey(family, ctx) {
				return empty, false
			}
			selected = directOwnerSpecializedOverrideBridgeSelection{family: family, bridge: bridge}
		}
	}
	return selected, selected.family != nil
}

// Frozen public232 function: genericFamilyMethodPlan; only capsule call names change.
func superclassQuery4059LegacyGenericFamilyMethodPlan(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) (*directOwnerOverrideBridgeFamilyPlan, bool) {
	if owner == nil || method == nil || method.IsPrivate || method.IsStatic || method.Constructor || len(method.TypeParameters) != 0 {
		return nil, false
	}
	layout := superclassQuery4059LegacyCanonicalGenericFamily(owner, ctx)
	if layout == nil {
		return nil, false
	}
	if len(methodDirectOwnerTypeParameterDeclarations(owner, method)) == 0 {
		return nil, false
	}
	plan := &directOwnerOverrideBridgeFamilyPlan{owner: owner, method: method, erasedResult: "void", requiresErasedView: true}
	for _, parameter := range method.Parameters {
		typ, _, ok := directOwnerOverrideBridgeErasedType(owner, parameter, ctx)
		if !ok {
			return nil, false
		}
		plan.erasedParameters = append(plan.erasedParameters, typ)
	}
	if !javaMethodResultIsVoid(method) {
		typ, _, ok := directOwnerOverrideBridgeErasedType(owner, method, ctx)
		if !ok {
			return nil, false
		}
		plan.erasedResult = typ
	}
	for descendant := range layout.members {
		if descendant == owner {
			continue
		}
		args := mapClassTypeArgumentStringsToAncestor(descendant, descendant.GoTypeParameterNames(), owner, ctx)
		if len(args) != len(owner.TypeParameters) {
			continue
		}
		parameters, result := mappedOverrideSourceSignature(owner, method, args)
		matches := directDeclaredOverrides(owner, descendant, method, parameters, result, ctx)
		for _, override := range matches {
			bridge, required, ok := planDirectOwnerSpecializedOverride(owner, method, descendant, override, args, parameters, result, plan.erasedParameters, plan.erasedResult, ctx)
			if !ok {
				return nil, false
			}
			if required {
				plan.overrides = append(plan.overrides, bridge)
			}
		}
	}
	return plan, true
}

// Frozen public232 function: canonicalGenericFamily; only capsule call names change.
func superclassQuery4059LegacyCanonicalGenericFamily(scope *symbol.ClassScope, ctx Ctx) *genericFamilyPlan {
	if scope == nil {
		return nil
	}
	var plans []*genericFamilyPlan
	if ctx.genericFamilies != nil {
		if !ctx.genericFamilies.ready {
			inventoryCtx := ctx.Clone()
			inventoryCtx.genericFamilies = nil
			ctx.genericFamilies.plans = superclassQuery4059LegacyDiscoverCanonicalGenericFamilies(inventoryCtx)
			ctx.genericFamilies.ready = true
		}
		plans = ctx.genericFamilies.plans
	} else {
		plans = superclassQuery4059LegacyDiscoverCanonicalGenericFamilies(ctx)
	}
	for _, plan := range plans {
		if _, member := plan.members[scope]; member {
			return plan
		}
		if scope.Class != nil && scope.Class.DeclarationNode != nil {
			file := superclassQuery4059LegacyFindFileScopeForClassScope(scope.Enclosing)
			if file == nil {
				file = ctx.currentFile
			}
			key := genericFamilyKey(file, scope.Class.DeclarationNode)
			_, anonymous := plan.anonymous[key]
			_, local := plan.locals[key]
			if anonymous || local {
				instance := *plan
				instance.members = make(map[*symbol.ClassScope]struct{}, len(plan.members)+1)
				for member := range plan.members {
					instance.members[member] = struct{}{}
				}
				instance.members[scope] = struct{}{}
				instance.binders = make(map[*symbol.TypeParamDeclaration]struct{}, len(plan.binders))
				for declaration := range plan.binders {
					instance.binders[declaration] = struct{}{}
				}
				instance.representations = make(map[*symbol.TypeParamDeclaration]genericFamilyBinderRepresentation, len(plan.representations))
				for declaration, representation := range plan.representations {
					instance.representations[declaration] = representation
				}
				if err := instance.addBinderRepresentations(scope, ctx); err != nil {
					return nil
				}
				return &instance
			}
		}
	}
	return nil
}

// Frozen public232 function: discoverCanonicalGenericFamilies; only capsule call names change.
func superclassQuery4059LegacyDiscoverCanonicalGenericFamilies(ctx Ctx) []*genericFamilyPlan {
	seen := map[*symbol.ClassScope]bool{}
	var plans []*genericFamilyPlan
	var inventory *genericFamilyInventory
	planSeed := func(seed *symbol.ClassScope) (*genericFamilyPlan, error) {
		if inventory == nil {
			inventory = newGenericFamilyInventory(ctx)
		}
		return planGenericFamilyWithInventory(seed, ctx, inventory)
	}
	for _, seed := range superclassQuery4059LegacySourceGenericViewDemandSeeds(ctx) {
		seen[seed] = true
		if plan, err := planSeed(seed); err == nil {
			plans = append(plans, plan)
		}
	}
	for _, owner := range allSourceClassScopes() {
		for _, method := range owner.Methods {
			if method == nil || len(method.TypeParameters) == 0 {
				continue
			}
			types := []string{method.OriginalType}
			for _, parameter := range method.Parameters {
				types = append(types, parameter.OriginalType)
			}
			for _, typ := range types {
				base, args := parseJavaTypeString(typ)
				if len(args) == 0 {
					continue
				}
				demanded := false
				for _, parameter := range method.TypeParameters {
					demanded = demanded || javaTypeContainsParameter(typ, parameter.Name)
				}
				if !demanded {
					continue
				}
				seed := resolveClassScopeByQualifiedName(superclassQuery4059LegacyClassScopeCtx(owner, ctx), base)
				if seed == nil || len(seed.TypeParameters) == 0 || leafObjectMemberErasureEligible(seed, ctx) {
					continue
				}
				if seen[seed] {
					continue
				}
				seen[seed] = true
				plan, err := planSeed(seed)
				if err != nil {
					continue
				}

				plans = append(plans, plan)
			}
		}
	}
	return plans
}

// Frozen public232 function: sourceGenericViewDemandSeeds; only capsule call names change.
func superclassQuery4059LegacySourceGenericViewDemandSeeds(ctx Ctx) []*symbol.ClassScope {
	if seeds, known := superclassQuery4059LegacySourceGenericViewDemandSeedsFromSyntax(ctx); known {
		return seeds
	}
	seen := map[*symbol.ClassScope]bool{}
	var seeds []*symbol.ClassScope
	for _, owner := range allSourceClassScopes() {
		if owner.Class == nil || owner.Class.DeclarationNode == nil {
			continue
		}
		file := superclassQuery4059LegacyFindFileScopeForClassScope(owner, ctx)
		if file == nil {
			continue
		}
		root := owner.Class.DeclarationNode
		var walk func(*sitter.Node, Ctx)
		walk = func(node *sitter.Node, lexical Ctx) {
			if node == nil {
				return
			}
			if superclassQuery4059LegacySourceGenericViewTypeDeclaration(node, ctx) && !node.Equal(root) {
				// Named classes are visited in their own declaration context. Local
				// class scopes are not in the resolved graph; do not resolve their
				// private binders or nested type names as enclosing declarations.
				return
			}
			if superclassQuery4059LegacyResolvedSourceNodeType(node, ctx) == "class_body" {
				parent := node.Parent()
				if parent != nil && (superclassQuery4059LegacyResolvedSourceNodeType(parent, ctx) == "object_creation_expression" || superclassQuery4059LegacyResolvedSourceNodeType(parent, ctx) == "enum_constant") {
					// Anonymous declarations are hoisted later. Their lexical type
					// namespace is not this resolved owner; do not invent a demand
					// by resolving an anonymous member's shadow as an outer type.
					return
				}
			}
			if superclassQuery4059LegacyResolvedSourceNodeType(node, ctx) == "method_declaration" || superclassQuery4059LegacyResolvedSourceNodeType(node, ctx) == "constructor_declaration" {
				lexical = lexical.Clone()
				lexical.localScope = nil
				for _, method := range owner.Methods {
					if method.DeclarationNode != nil && method.DeclarationNode.Equal(node) {
						lexical.localScope = method
						break
					}
				}
			}
			if seed := superclassQuery4059LegacySourceGenericViewDemandType(node, lexical, file.Source); seed != nil && !seen[seed] {
				seen[seed] = true
				seeds = append(seeds, seed)
			}
			for _, child := range superclassQuery4059LegacyResolvedSourceNamedChildren(node, ctx) {
				childCtx := lexical
				// The original helper returns this context for non-declarations.
				// Reuse its structural guard while keeping header/body selection fresh.
				if superclassQuery4059LegacySourceGenericViewTypeDeclaration(node, ctx) {
					childCtx = classDeclarationChildCtx(node, child, lexical)
				}
				walk(child, childCtx)
			}
		}
		walk(root, superclassQuery4059LegacyClassScopeCtx(owner, ctx))
	}
	// The project symbol graph is map-backed. Declaration identity selects the
	// layout; sorting prevents file-render order from selecting a different seed.
	sort.Slice(seeds, func(i, j int) bool {
		return qualifiedSourceClassName(seeds[i]) < qualifiedSourceClassName(seeds[j])
	})
	return seeds
}

// Frozen public232 function: sourceGenericViewDemandType; only capsule call names change.
func superclassQuery4059LegacySourceGenericViewDemandType(node *sitter.Node, ctx Ctx, source []byte) *symbol.ClassScope {
	switch superclassQuery4059LegacyResolvedSourceNodeType(node, ctx) {
	case "generic_type", "type_identifier", "scoped_type_identifier":
	default:
		return nil
	}
	parent := node.Parent()
	if parent != nil {
		if superclassQuery4059LegacyResolvedSourceNodeType(parent, ctx) == "scoped_type_identifier" {
			return nil // resolve the complete qualified spelling only
		}
		if superclassQuery4059LegacyResolvedSourceNodeType(parent, ctx) == "generic_type" && superclassQuery4059LegacyResolvedSourceNodeType(node, ctx) != "generic_type" {
			return nil // the base of Cell<String> is not a raw Cell occurrence
		}
	}
	javaType := node.Content(source)
	if genericFamilyCanonicalComparableHeader(javaType, ctx, node) {
		return ctx.currentClass
	}
	base, arguments := parseJavaTypeString(javaType)
	if visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || superclassQuery4059LegacySourceGenericViewLocalTypeShadowed(node, base, source) {
		return nil
	}
	demanded := superclassQuery4059LegacySourceGenericViewCastType(node)
	if superclassQuery4059LegacyResolvedSourceNodeType(node, ctx) == "generic_type" {
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

// Frozen public232 function: sourceGenericViewCastType; only capsule call names change.
func superclassQuery4059LegacySourceGenericViewCastType(node *sitter.Node) bool {
	for parent := node.Parent(); parent != nil; parent = node.Parent() {
		if parent.Type() == "array_type" || parent.Type() == "annotated_type" {
			node = parent
			continue
		}
		return parent.Type() == "cast_expression" && sameSourceNode(node, parent.ChildByFieldName("type"))
	}
	return false
}

// Frozen public232 function: sourceGenericViewTypeDeclaration; only capsule call names change.
func superclassQuery4059LegacySourceGenericViewTypeDeclaration(node *sitter.Node, contexts ...Ctx) bool {
	var ctx Ctx
	if len(contexts) > 0 {
		ctx = contexts[0]
	}
	if node == nil {
		return false
	}
	switch superclassQuery4059LegacyResolvedSourceNodeType(node, ctx) {
	case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration":
		return true
	}
	return false
}

// Frozen public232 function: sourceGenericViewLocalTypeShadowed; only capsule call names change.
func superclassQuery4059LegacySourceGenericViewLocalTypeShadowed(node *sitter.Node, base string, source []byte) bool {
	name := strings.SplitN(strings.TrimSpace(base), ".", 2)[0]
	declaredBefore := func(child *sitter.Node) bool {
		if child == nil || child.StartByte() >= node.StartByte() {
			return false
		}
		if superclassQuery4059LegacySourceGenericViewTypeDeclaration(child) {
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

// Frozen public232 function: sourceGenericViewDemandSeedsFromSyntax; only capsule call names change.
func superclassQuery4059LegacySourceGenericViewDemandSeedsFromSyntax(ctx Ctx) ([]*symbol.ClassScope, bool) {
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
		base := superclassQuery4059LegacyClassScopeCtx(owner, ctx)
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
				if seed := superclassQuery4059LegacySourceGenericViewDemandType(node, lexical, entry.file.Source); seed != nil && !seen[seed] {
					seen[seed] = true
					seeds = append(seeds, seed)
				}
			}
		}
	}
	sort.Slice(seeds, func(i, j int) bool { return qualifiedSourceClassName(seeds[i]) < qualifiedSourceClassName(seeds[j]) })
	return seeds, true
}

// Frozen public232 function: resolvedSourceNodeType; only capsule call names change.
func superclassQuery4059LegacyResolvedSourceNodeType(node *sitter.Node, ctx Ctx) string {
	if node == nil {
		return ""
	}
	inventory := superclassQuery4059LegacyResolvedSourceInventory(ctx)
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

// Frozen public232 function: resolvedSourceNamedChildren; only capsule call names change.
func superclassQuery4059LegacyResolvedSourceNamedChildren(node *sitter.Node, ctx Ctx) []*sitter.Node {
	inventory := superclassQuery4059LegacyResolvedSourceInventory(ctx)
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

// Frozen public232 function: resolvedSourceInventory; only capsule call names change.
func superclassQuery4059LegacyResolvedSourceInventory(ctx Ctx) *callableSubclassSourceInventory {
	inventory := ctx.callableSubclasses
	if inventory != nil && inventory.graph != symbol.GlobalScope {
		// All facts have the same graph lifetime, including negative/empty syntax.
		retireStaleDeclarationInvocation(ctx)
		if inventory.ownership != nil {
			inventory.ownership.declarations.retire()
		}
		*inventory = callableSubclassSourceInventory{graph: symbol.GlobalScope}
	}
	return inventory
}

// Frozen public232 function: classScopeCtx; only capsule call names change.
func superclassQuery4059LegacyClassScopeCtx(scope *symbol.ClassScope, ctx Ctx) Ctx {
	result := ctx.Clone()
	result.emittedReceiverOwner = nil
	result.emittedReceiverName = ""
	result.currentClass = scope
	result.memberTypeHeaderOwner = nil
	result.localScope = nil
	result.localBindingBody = nil
	for current := scope; current != nil; current = current.Enclosing {
		if file := superclassQuery4059LegacyFindFileScopeForClassScope(current, ctx); file != nil {
			result.currentFile = file
			break
		}
	}
	return result
}

// Frozen public232 function: findFileScopeForClassScope; only capsule call names change.
func superclassQuery4059LegacyFindFileScopeForClassScope(scope *symbol.ClassScope, contexts ...Ctx) *symbol.FileScope {
	if scope == nil {
		return nil
	}
	if activeResolutionFiles != nil && activeResolutionFiles.graph == symbol.GlobalScope {
		if file := activeResolutionFiles.files[scope]; file != nil {
			return file
		}
	}
	if len(contexts) > 0 {
		if ownership := sourceOwnershipIndex(contexts[0]); ownership != nil {
			if file := ownership.files[scope]; file != nil {
				return file
			}
		}
	}
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			for _, top := range file.TopLevelClasses {
				if classScopeContains(top, scope) {
					return file
				}
			}
		}
	}
	return nil
}
