package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// A nonstatic lexical class chain carries enclosing class declarations even
// though that chain is not Java inheritance. This sidecar recovers only those
// original class slots; executable captures keep their separate owner proof.
func declarationCapturedClassTypeOrigins(scope, receiver *symbol.ClassScope, arguments []inferredJavaTypeOrigin, ctx Ctx) map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin {
	if scope == nil || receiver == nil || receiver != ctx.currentClass || ctx.currentFile == nil || scope.Class == nil || receiver.Class == nil || scope.Class.DeclarationNode == nil || receiver.Class.DeclarationNode == nil || !scope.IsInner || scope.Enclosing == nil || len(arguments) != len(receiver.TypeParameters) {
		return nil
	}
	if ctx.localScope != nil && (ctx.localScope.IsStatic || !threadSourceNodeWithin(ctx.localScope.DeclarationNode, receiver.Class.DeclarationNode)) {
		return nil
	}
	hidden := len(scope.TypeParameters) - len(scope.OwnTypeParameters())
	if hidden <= 0 {
		return nil
	}
	owned := func(candidate *symbol.ClassScope) bool {
		if candidate == nil || candidate.Class == nil || candidate.Class.DeclarationNode == nil {
			return false
		}
		if findFileScopeForClassScope(candidate, ctx) == ctx.currentFile {
			return true
		}
		for _, local := range ctx.localClasses {
			if local != nil && local.scope == candidate {
				return true
			}
		}
		return false
	}
	ancestor := scope.Enclosing
	if !owned(scope) || !owned(receiver) || !owned(ancestor) || !threadSourceNodeWithin(scope.Class.DeclarationNode, ancestor.Class.DeclarationNode) {
		return nil
	}
	seen := map[*symbol.ClassScope]bool{}
	for current := receiver; ; current = current.Enclosing {
		if current == nil || seen[current] || !owned(current) {
			return nil
		}
		seen[current] = true
		if current == ancestor {
			break
		}
		if !current.IsInner || current.Enclosing == nil || current.Enclosing.Class == nil || !threadSourceNodeWithin(current.Class.DeclarationNode, current.Enclosing.Class.DeclarationNode) {
			return nil
		}
	}
	// Only declarations written on lexical classes belong to this fallback.
	// A method binder carried by a local class must use the executable proof.
	classes := map[*symbol.TypeParamDeclaration]bool{}
	seen = map[*symbol.ClassScope]bool{}
	for current := ancestor; current != nil; current = current.Enclosing {
		if seen[current] || !owned(current) {
			return nil
		}
		seen[current] = true
		for _, parameter := range current.OwnTypeParameters() {
			if parameter.Declaration != nil {
				classes[parameter.Declaration] = true
			}
		}
		if !current.IsInner {
			break
		}
		if current.Enclosing == nil || current.Enclosing.Class == nil || !threadSourceNodeWithin(current.Class.DeclarationNode, current.Enclosing.Class.DeclarationNode) {
			return nil
		}
	}
	var result map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin
	for _, captured := range scope.TypeParameters[:hidden] {
		declaration := captured.Declaration
		if declaration == nil || !classes[declaration] {
			continue
		}
		found := -1
		for index, parameter := range receiver.TypeParameters {
			if parameter.Declaration == declaration {
				if found != -1 {
					return nil
				}
				found = index
			}
		}
		if found == -1 {
			continue
		}
		value := arguments[found]
		if !completeJavaTypeOrigin(value) || value.wildcard || value.parameter != nil && value.parameter != declaration {
			continue
		}
		if result == nil {
			result = make(map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin)
		}
		result[declaration] = value
	}
	return result
}
