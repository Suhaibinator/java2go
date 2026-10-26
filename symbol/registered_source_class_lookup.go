package symbol

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func registeredSourceRoots(file *FileScope) []*ClassScope {
	if file == nil {
		return nil
	}
	if len(file.TopLevelClasses) != 0 {
		return file.TopLevelClasses
	}
	if file.BaseClass != nil {
		return []*ClassScope{file.BaseClass}
	}
	return nil
}

func registeredSourceScopes(roots []*ClassScope, visit func(*ClassScope)) {
	seen := make(map[*ClassScope]bool)
	var walk func(*ClassScope)
	walk = func(scope *ClassScope) {
		if scope == nil || scope.Class == nil || seen[scope] {
			return
		}
		seen[scope] = true
		visit(scope)
		for _, nested := range scope.Subclasses {
			walk(nested)
		}
	}
	for _, root := range roots {
		walk(root)
	}
}

// A repeated scope is one declaration; distinct matches are ambiguous even if
// map iteration happens to encounter one first. Matching uses Java identity.
func registeredSourceClassMatch(roots []*ClassScope, name string, recursive bool) (*ClassScope, bool) {
	var found *ClassScope
	ambiguous := false
	match := func(scope *ClassScope) {
		if scope == nil || scope.Class == nil || name == "" || scope.Class.OriginalName != name {
			return
		}
		if found != nil && found != scope {
			ambiguous = true
		}
		found = scope
	}
	if recursive {
		registeredSourceScopes(roots, match)
	} else {
		for _, root := range roots {
			match(root)
		}
	}
	if ambiguous {
		return nil, true
	}
	return found, false
}

func registeredSourceNodeWithin(node, owner *sitter.Node) bool {
	if node == nil || owner == nil {
		return false
	}
	for current := node; current != nil; current = current.Parent() {
		if current.Equal(owner) {
			return true
		}
	}
	return false
}

func registeredSourceDefinitionContains(member, target *Definition) bool {
	seen := make(map[*Definition]bool)
	var walk func(*Definition) bool
	walk = func(definition *Definition) bool {
		if definition == nil || seen[definition] {
			return false
		}
		seen[definition] = true
		if definition == target {
			return true
		}
		for _, parameter := range definition.Parameters {
			if walk(parameter) {
				return true
			}
		}
		for _, child := range definition.Children {
			if walk(child) {
				return true
			}
		}
		return false
	}
	return walk(member)
}

// Parameters need not have their own AST node. Exact membership in the original
// declared method, whose node belongs to this class, still proves their owner.
func registeredSourceDefinitionOwner(definition *Definition, roots []*ClassScope) (*ClassScope, bool) {
	var found *ClassScope
	ambiguous := false
	registeredSourceScopes(roots, func(scope *ClassScope) {
		members := append([]*Definition(nil), scope.Fields...)
		members = append(members, scope.Methods...)
		for _, member := range members {
			if member == nil || !registeredSourceNodeWithin(member.DeclarationNode, scope.Class.DeclarationNode) || !registeredSourceDefinitionContains(member, definition) {
				continue
			}
			if found != nil && found != scope {
				ambiguous = true
			}
			found = scope
		}
	})
	if ambiguous {
		return nil, true
	}
	return found, false
}

// Only an actual lexical owner can make a nested declaration shadow an import.
// Top-level declarations are file-wide; unrelated siblings' members are not.
func registeredSourceLocalClass(definition *Definition, file *FileScope, name string) (*ClassScope, bool, bool) {
	roots := registeredSourceRoots(file)
	owner, ambiguous := registeredSourceDefinitionOwner(definition, roots)
	if ambiguous {
		return nil, false, true
	}
	if owner != nil {
		members := make(map[*ClassScope]bool)
		registeredSourceScopes(roots, func(scope *ClassScope) { members[scope] = true })
		var chain []*ClassScope
		seen := make(map[*ClassScope]bool)
		for current := owner; current != nil; current = current.Enclosing {
			if seen[current] || !members[current] || current.Class.DeclarationNode == nil {
				return nil, false, true
			}
			if len(chain) > 0 && !registeredSourceNodeWithin(chain[len(chain)-1].Class.DeclarationNode, current.Class.DeclarationNode) {
				return nil, false, true
			}
			seen[current] = true
			chain = append(chain, current)
		}
		rooted := false
		for _, root := range roots {
			rooted = rooted || root == chain[len(chain)-1]
		}
		if !rooted {
			return nil, false, true
		}
		for _, current := range chain {
			candidates := append([]*ClassScope{current}, current.Subclasses...)
			if found, multiple := registeredSourceClassMatch(candidates, name, false); found != nil || multiple {
				return found, true, multiple
			}
		}
	}
	found, multiple := registeredSourceClassMatch(roots, name, false)
	if owner == nil {
		// Without lexical ownership a nested homonym cannot be ruled out.
		var nested []*ClassScope
		for _, root := range roots {
			if root != nil {
				nested = append(nested, root.Subclasses...)
			}
		}
		if possible, repeated := registeredSourceClassMatch(nested, name, true); possible != nil || repeated {
			return nil, false, true
		}
	}
	return found, owner != nil, multiple
}

func registeredSourceMemberPath(scope *ClassScope, names []string) *ClassScope {
	seen := make(map[*ClassScope]bool)
	for _, name := range names {
		if scope == nil || seen[scope] {
			return nil
		}
		seen[scope] = true
		var ambiguous bool
		scope, ambiguous = registeredSourceClassMatch(scope.Subclasses, name, false)
		if ambiguous || scope != nil && seen[scope] {
			return nil
		}
	}
	return scope
}

// An import owner can be a package or a member-type prefix. Every accepted path
// must start at an actual registered package and follow its declared members.
func registeredSourceQualifiedClass(name string) *ClassScope {
	if GlobalScope == nil {
		return nil
	}
	parts := strings.Split(name, ".")
	var found *ClassScope
	for split := 1; split < len(parts); split++ {
		pkg := GlobalScope.FindPackage(strings.Join(parts[:split], "."))
		if pkg == nil {
			continue
		}
		var roots []*ClassScope
		for _, file := range pkg.Files {
			roots = append(roots, registeredSourceRoots(file)...)
		}
		candidate, ambiguous := registeredSourceClassMatch(roots, parts[split], false)
		if ambiguous {
			return nil
		}
		candidate = registeredSourceMemberPath(candidate, parts[split+1:])
		if candidate == nil {
			continue
		}
		if found != nil && found != candidate {
			return nil
		}
		found = candidate
	}
	return found
}

func registeredSourceDefinitionClass(definition *Definition, fileScope *FileScope) *ClassScope {
	if definition == nil || fileScope == nil || definition.OriginalType == "" || definition.DirectTypeParameter != nil {
		return nil
	}
	parts := strings.Split(definition.OriginalType, ".")
	for _, part := range parts {
		if part == "" {
			return nil
		}
	}
	if definition.TypeParameterBindings[parts[0]] != nil {
		return nil
	}
	selected, owned, refused := registeredSourceLocalClass(definition, fileScope, parts[0])
	if refused {
		return nil
	}
	if selected != nil {
		selected = registeredSourceMemberPath(selected, parts[1:])
	} else {
		if !owned {
			return nil
		}
		if imported, exists := fileScope.Imports[parts[0]]; exists {
			selected = registeredSourceQualifiedClass(imported + "." + parts[0])
			selected = registeredSourceMemberPath(selected, parts[1:])
		} else if len(parts) > 1 {
			selected = registeredSourceQualifiedClass(definition.OriginalType)
		}
	}
	return selected
}
