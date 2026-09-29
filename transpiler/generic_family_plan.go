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
	members         map[*symbol.ClassScope]struct{}
	anonymous       map[genericFamilySourceKey]genericFamilyAnonymous
	locals          map[genericFamilySourceKey]genericFamilyAnonymous
	binders         map[*symbol.TypeParamDeclaration]struct{}
	representations map[*symbol.TypeParamDeclaration]genericFamilyBinderRepresentation
}

type genericFamilySourceKey struct {
	file       *symbol.FileScope
	start, end uint32
}

type genericFamilyAnonymous struct {
	owner    *symbol.ClassScope
	parent   *symbol.ClassScope
	node     *sitter.Node
	javaType string
}

func genericFamilyKey(file *symbol.FileScope, node *sitter.Node) genericFamilySourceKey {
	return genericFamilySourceKey{file: file, start: node.StartByte(), end: node.EndByte()}
}

func planGenericFamily(seed *symbol.ClassScope, ctx Ctx) (*genericFamilyPlan, error) {
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
	scopes := allSourceClassScopes()
	localInterfaces := genericFamilyLocalInterfaceEdges(scopes, ctx)
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
			parents := genericFamilyParents(scope, ctx)
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
	// Scan each physical source file once, preserving the lexical owner and
	// method binder environment. Anonymous classes are inventoried from source,
	// not from the order in which expression rendering happens to discover them.
	for _, owner := range scopes {
		if owner.Class == nil || owner.Class.DeclarationNode == nil {
			continue
		}
		file := findFileScopeForClassScope(owner)
		if file == nil {
			return nil, fmt.Errorf("missing source for generic family inventory")
		}
		ownerCtx := classScopeCtx(owner, ctx)
		var scan func(*sitter.Node, Ctx) error
		scan = func(node *sitter.Node, lexical Ctx) error {
			if node == nil {
				return nil
			}
			if node != owner.Class.DeclarationNode && (node.Type() == "class_declaration" || node.Type() == "interface_declaration" || node.Type() == "enum_declaration") {
				// Named member declarations have their own resolved ClassScope. Local
				// classes do not, so they must be diagnosed if they extend this family.
				parent := node.Parent()
				if parent != nil && parent.Type() != "class_body" && parent.Type() != "program" {
					if interfaces := node.ChildByFieldName("interfaces"); interfaces != nil {
						for _, typ := range collectTypeNodes(interfaces) {
							javaType := typ.Content(file.Source)
							base, _ := parseJavaTypeString(javaType)
							parent := resolveClassScopeByQualifiedName(lexical, base)
							if _, member := plan.members[parent]; !member {
								continue
							}
							body := node.ChildByFieldName("body")
							parameters := planLocalClassTypeParameters(node, nil, file.Source, lexical)
							local := synthLocalClassScope(node, body, node.ChildByFieldName("name").Content(file.Source), "", nil, nil, nil, parameters.carried, parameters.declared, anonymousClassMethods(body), localClassConstructors(body), nil, file.Source)
							local.Enclosing = owner
							localCtx := lexical.Clone()
							localCtx.currentClass = local
							localCtx.localScope = nil
							if err := plan.addBinderRepresentations(local, localCtx); err != nil {
								return err
							}
							if !plan.typeSupported(javaType, localCtx, true) {
								return fmt.Errorf("local generic family specialization needs shared storage: %s", javaType)
							}
							if err := plan.auditTypeSyntax(node, file.Source, localCtx); err != nil {
								return err
							}
							plan.locals[genericFamilyKey(file, node)] = genericFamilyAnonymous{owner: owner, parent: parent, node: node, javaType: javaType}
						}
					}
					if superclass := node.ChildByFieldName("superclass"); superclass != nil {
						for _, typ := range collectTypeNodes(superclass) {
							base, _ := parseJavaTypeString(typ.Content(file.Source))
							if _, member := plan.members[resolveClassScopeByQualifiedName(lexical, base)]; member {
								return fmt.Errorf("local subclass requires a planned source scope")
							}
						}
					}
				}
				return nil
			}
			if node.Type() == "method_declaration" || node.Type() == "constructor_declaration" {
				for _, method := range owner.Methods {
					if method.DeclarationNode != nil && method.DeclarationNode.StartByte() == node.StartByte() && method.DeclarationNode.EndByte() == node.EndByte() {
						lexical = lexical.Clone()
						lexical.localScope = method
						break
					}
				}
			}
			if node.Type() == "object_creation_expression" {
				typ := node.ChildByFieldName("type")
				if typ != nil {
					base, _ := parseJavaTypeString(typ.Content(file.Source))
					parent := resolveClassScopeByQualifiedName(lexical, base)
					if _, member := plan.members[parent]; member {
						for _, child := range nodeutil.NamedChildrenOf(node) {
							if child.Type() == "class_body" {
								key := genericFamilyKey(file, node)
								plan.anonymous[key] = genericFamilyAnonymous{owner: owner, parent: parent, node: node, javaType: typ.Content(file.Source)}
								if !plan.typeSupported(typ.Content(file.Source), lexical, true) {
									return fmt.Errorf("anonymous generic family specialization needs shared storage: %s", typ.Content(file.Source))
								}
								if err := plan.auditTypeSyntax(child, file.Source, lexical); err != nil {
									return err
								}
							}
						}
					}
				}
			}
			for _, child := range nodeutil.NamedChildrenOf(node) {
				if err := scan(child, lexical); err != nil {
					return err
				}
			}
			return nil
		}
		if err := scan(owner.Class.DeclarationNode, ownerCtx); err != nil {
			return nil, err
		}
	}
	audited := map[*symbol.ClassScope]bool{}
	for len(audited) < len(plan.members) {
		for scope := range plan.members {
			if audited[scope] {
				continue
			}
			audited[scope] = true
			declarationCtx := classScopeCtx(scope, ctx)
			for _, method := range scope.Methods {
				if method == nil || !method.Constructor {
					continue
				}
				for _, parameter := range scope.TypeParameters {
					if !constructorBodySupportsCallableErasure(scope, method, parameter.Declaration, ctx) {
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
				if !plan.typeSupported(parent, declarationCtx, true) {
					return nil, fmt.Errorf("generic family specialization needs shared storage: %s", parent)
				}
			}
			file := findFileScopeForClassScope(scope)
			if file == nil {
				return nil, fmt.Errorf("missing generic family declaration source")
			}
			if err := plan.auditTypeSyntax(scope.Class.DeclarationNode, file.Source, declarationCtx); err != nil {
				return nil, err
			}
		}
	}
	return plan, nil
}

func genericFamilyParents(scope *symbol.ClassScope, ctx Ctx) []*symbol.ClassScope {
	if scope == nil {
		return nil
	}
	var result []*symbol.ClassScope
	declarationCtx := classScopeCtx(scope, ctx)
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
func (plan *genericFamilyPlan) typeSupported(typ string, ctx Ctx, specialization bool) bool {
	typ = strings.TrimSpace(typ)
	if typ == "" {
		return true
	}
	for _, prefix := range []string{"? extends ", "? super "} {
		typ = strings.TrimPrefix(typ, prefix)
	}
	component, rank := javaArrayTypeParts(typ)
	base, args := parseJavaTypeString(component)
	declaration := visibleTypeParameterDeclarationForJavaType(base, ctx)
	_, owned := plan.binders[declaration]
	if len(args) == 0 {
		return !owned || rank == 0
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
	target := resolveClassScopeByQualifiedName(ctx, base)
	_, member := plan.members[target]
	if !member {
		if !containsOwned || !genericFamilyStorageLeaf(target, ctx) {
			return false
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
		if !plan.typeSupported(argument, ctx, true) {
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
			if !plan.typeSupported(node.Content(source), lexical, false) {
				return fmt.Errorf("generic family type requires unsupported physical storage: %s", node.Content(source))
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			if err := walk(child, lexical); err != nil {
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
		file := findFileScopeForClassScope(owner)
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
