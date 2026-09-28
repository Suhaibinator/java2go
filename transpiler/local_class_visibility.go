package transpiler

import sitter "github.com/smacker/go-tree-sitter"

// Hoisted declarations share a render registry, but their Java names remain
// scoped to the declaring executable. In particular, a later method or a
// whole-project type inventory must not inherit a previous method's bindings.
func localClassInDeclaration(name string, ctx Ctx) *localClassInfo {
	info := ctx.localClasses[name]
	if info == nil || info.scope == nil || info.scope.Class == nil {
		return nil
	}
	declaration := info.scope.Class.DeclarationNode
	var executable *sitter.Node
	for node := declaration; node != nil; node = node.Parent() {
		if node.Type() == "method_declaration" || node.Type() == "constructor_declaration" {
			executable = node
			break
		}
	}
	if executable == nil {
		return info
	}
	inside := func(node *sitter.Node) bool {
		for ; node != nil; node = node.Parent() {
			if node.Equal(executable) {
				return true
			}
		}
		return false
	}
	if ctx.localScope != nil && inside(ctx.localScope.DeclarationNode) {
		return info
	}
	if ctx.currentClass != nil && ctx.currentClass.Class != nil && inside(ctx.currentClass.Class.DeclarationNode) {
		return info
	}
	return nil
}

// A block starts with the visible outer bindings and owns every subsequent
// declaration or shadowing update. Hoisted declarations themselves stay in the
// shared output list, independently of this lexical lookup map.
func copyLocalClassBindings(outer map[string]*localClassInfo) map[string]*localClassInfo {
	bindings := make(map[string]*localClassInfo, len(outer))
	for name, info := range outer {
		bindings[name] = info
	}
	return bindings
}
