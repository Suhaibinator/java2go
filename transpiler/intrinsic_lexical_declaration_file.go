package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// Late local/anonymous scopes are not entries in the resolved file index. Their
// original source tree and live registry can certify the same lexical file,
// without admitting a detached declaration through an enclosing name or span.
func intrinsicLexicalDeclaringFile(scope *symbol.ClassScope, file *symbol.FileScope, ctx Ctx) bool {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil || file == nil || ctx.currentFile != file {
		return false
	}
	node := scope.Class.DeclarationNode
	registered := false
	switch node.Type() {
	case "object_creation_expression":
		if key, ok := anonymousClassSourceKey(node); ok {
			info := ctx.anonymousClasses[key]
			registered = info != nil && info.scope == scope && info.scope.Class.DeclarationNode.Equal(node)
		}
	case "class_declaration":
		for _, info := range ctx.localClasses {
			if info != nil && info.scope == scope && info.scope.Class.DeclarationNode.Equal(node) {
				registered = true
				break
			}
		}
	}
	if !registered {
		return false
	}
	seen := map[*symbol.ClassScope]bool{scope: true}
	child := scope
	derived := false
	for owner := scope.Enclosing; owner != nil; owner = owner.Enclosing {
		if seen[owner] || owner.Class == nil || owner.Class.DeclarationNode == nil || !threadSourceNodeWithin(child.Class.DeclarationNode, owner.Class.DeclarationNode) {
			return false
		}
		seen[owner] = true
		if registeredFile := findFileScopeForClassScope(owner, ctx); registeredFile != nil {
			if registeredFile != file {
				return false
			}
			derived = true
		}
		child = owner
	}
	return derived
}
