package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Reader dispatch depends on the receiver's real source declaration. Generic
// spelling and the caller's imports cannot donate its native superclass edge.
func readerSourceInvocationOwner(object *sitter.Node, ctx Ctx, source []byte) (*symbol.ClassScope, bool) {
	if object == nil || ctx.currentFile == nil || ctx.currentFile.ParsedRoot() == nil || !readerSourceNodeWithin(object, ctx.currentFile.ParsedRoot()) {
		return nil, false
	}
	var origin inferredJavaTypeOrigin
	if _, known := inferExprJavaType(object, ctx, source, &origin); !known || origin.nominalScope == nil || origin.parameter != nil || origin.arrayRank != 0 || origin.wildcard || !readerSourceCompleteOrigin(origin, ctx) {
		return nil, false
	}
	owner := origin.nominalScope
	seen := map[*symbol.ClassScope]bool{}
	for current := owner; current != nil && !seen[current]; {
		seen[current] = true
		declaring, owned := readerSourceDeclarationContext(current, ctx)
		if !owned || current.IsInterface || current.IsEnum {
			return nil, false
		}
		if parent := resolveSuperclassScopeInDeclaringContext(declaring, current); parent != nil {
			current = parent
			continue
		}
		native := declaredJavaTypeOrigin(symbol.JavaType{
			Original:              current.Superclass,
			TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(declaring)),
		}, declaring)
		if !completeJavaTypeOrigin(native) || native.nominalScope != nil || native.parameter != nil || native.arrayRank != 0 || native.wildcard || len(native.arguments) != 0 || native.externalName != "java.io.Reader" {
			return nil, false
		}
		return owner, true
	}
	return nil, false
}

func readerSourceDeclarationContext(owner *symbol.ClassScope, ctx Ctx) (Ctx, bool) {
	if owner == nil || owner.Class == nil || owner.Class.DeclarationNode == nil {
		return Ctx{}, false
	}
	file := findFileScopeForClassScope(owner, ctx)
	if file == nil || file.ParsedRoot() == nil || !readerSourceNodeWithin(owner.Class.DeclarationNode, file.ParsedRoot()) {
		return Ctx{}, false
	}
	declaring := ctx.Clone()
	declaring.currentFile = nil
	declaring.syntheticTypeParameters = nil
	declaring.localClasses = nil
	declaring = classHeaderTypeCtx(owner, declaring)
	if declaring.currentFile != file {
		return Ctx{}, false
	}
	return declaring, true
}

func readerSourceNodeWithin(node, owner *sitter.Node) bool {
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

// Parameters inside a known nominal receiver are legal captured declarations.
// A parameter at the receiver root is deliberately not certified here.
func readerSourceCompleteOrigin(origin inferredJavaTypeOrigin, ctx Ctx) bool {
	if !completeJavaTypeOrigin(origin) {
		return false
	}
	if origin.nominalScope != nil {
		if _, owned := readerSourceDeclarationContext(origin.nominalScope, ctx); !owned {
			return false
		}
	}
	if origin.parameter != nil {
		registered := false
		for _, owner := range allSourceClassScopes() {
			declaring, owned := readerSourceDeclarationContext(owner, ctx)
			if !owned {
				continue
			}
			for _, parameter := range owner.TypeParameters {
				registered = registered || parameter.Declaration == origin.parameter
			}
			for _, method := range owner.Methods {
				if method == nil || !readerSourceNodeWithin(method.DeclarationNode, owner.Class.DeclarationNode) || !readerSourceNodeWithin(method.DeclarationNode, declaring.currentFile.ParsedRoot()) {
					continue
				}
				for _, parameter := range method.TypeParameters {
					registered = registered || parameter.Declaration == origin.parameter
				}
			}
		}
		if !registered {
			return false
		}
	}
	for _, argument := range origin.arguments {
		if !readerSourceCompleteOrigin(argument, ctx) {
			return false
		}
	}
	return true
}
