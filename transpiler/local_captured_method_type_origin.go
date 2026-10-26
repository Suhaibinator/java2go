package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// A local class can carry an enclosing executable's binder independently of
// its enclosing class binders. Keep that original declaration pointer; a caller
// parameter with the same spelling supplies no evidence about this capture.
func localCapturedMethodTypeOrigin(scope *symbol.ClassScope, declaration *symbol.TypeParamDeclaration, ctx Ctx) (inferredJavaTypeOrigin, bool) {
	if scope == nil || scope.Class == nil || declaration == nil || scope.Class.DeclarationNode == nil {
		return inferredJavaTypeOrigin{}, false
	}
	node := scope.Class.DeclarationNode
	if node.Type() != "class_declaration" && node.Type() != "object_creation_expression" {
		return inferredJavaTypeOrigin{}, false
	}
	hidden := len(scope.TypeParameters) - len(scope.OwnTypeParameters())
	captured := false
	for index := 0; index < hidden; index++ {
		if scope.TypeParameters[index].Declaration == declaration {
			captured = true
			break
		}
	}
	if !captured {
		return inferredJavaTypeOrigin{}, false
	}
	for executable := node.Parent(); executable != nil; executable = executable.Parent() {
		if executable.Type() != "method_declaration" && executable.Type() != "constructor_declaration" {
			continue
		}
		inContext := ctx.localScope != nil && threadSourceNodeWithin(ctx.localScope.DeclarationNode, executable)
		if !inContext && ctx.currentClass != nil && ctx.currentClass.Class != nil {
			inContext = threadSourceNodeWithin(ctx.currentClass.Class.DeclarationNode, executable)
		}
		if !inContext {
			continue
		}
		for owner := scope.Enclosing; owner != nil; owner = owner.Enclosing {
			if owner.Class == nil || !threadSourceNodeWithin(node, owner.Class.DeclarationNode) {
				continue
			}
			file := findFileScopeForClassScope(owner, ctx)
			if file == nil {
				continue
			}
			for _, method := range owner.Methods {
				if method == nil || method.DeclarationNode == nil || !method.DeclarationNode.Equal(executable) {
					continue
				}
				for _, parameter := range method.TypeParameters {
					if parameter.Declaration == declaration {
						return inferredJavaTypeOrigin{declaringOwner: owner, parameter: declaration, javaType: parameter.EmittedName()}, true
					}
				}
			}
		}
	}
	return inferredJavaTypeOrigin{}, false
}
