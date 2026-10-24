package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
)

const threadSourceBundleMethod = "Java2goThreadSourceMethods"

// File authority follows actual AST ancestry, including synthetic declarations.
// Equal compares the tree identity as well as the node, never just byte spans.
func threadSourceNodeWithin(node, ancestor *sitter.Node) bool {
	if node == nil || ancestor == nil {
		return false
	}
	for current := node; current != nil; current = current.Parent() {
		if current.Equal(ancestor) {
			return true
		}
	}
	return false
}
func threadSourceDeclarationCtx(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) (Ctx, bool) {
	if owner == nil || owner.Class == nil {
		return ctx, false
	}
	file := findFileScopeForClassScope(owner, ctx)
	if file == nil {
		for parent := owner.Enclosing; parent != nil; parent = parent.Enclosing {
			if parent.Class != nil && threadSourceNodeWithin(owner.Class.DeclarationNode, parent.Class.DeclarationNode) {
				if actual := findFileScopeForClassScope(parent, ctx); actual != nil {
					file = actual
					break
				}
			}
		}
	}
	if file == nil {
		return ctx, false
	}
	declaring := ctx.Clone()
	declaring.currentFile = file
	declaring.syntheticTypeParameters = nil
	declaring.localClasses = nil
	node := owner.Class.DeclarationNode
	lexical := node != nil && (node.Type() == "object_creation_expression" || node.Parent() != nil && (node.Parent().Type() == "block" || node.Parent().Type() == "constructor_body"))
	if lexical {
		declaring.localClasses = make(map[string]*localClassInfo)
		for _, info := range ctx.localClasses {
			if info == nil || info.scope == nil || info.scope.Class == nil {
				continue
			}
			declaration := info.scope.Class.DeclarationNode
			if declaration == nil || declaration.Type() != "class_declaration" {
				continue
			}
			// A local type is visible in its original lexical block from its declaration.
			if !declaration.Equal(node) && (declaration.StartByte() > node.StartByte() || !threadSourceNodeWithin(node, declaration.Parent())) {
				continue
			}
			if name := declaration.ChildByFieldName("name"); name != nil {
				declaring.localClasses[name.Content(file.Source)] = info
			}
		}
	}
	declaring = classScopeCtx(owner, declaring)
	declaring.currentFile = file
	declaring.localScope = method
	return declaring, true
}
func threadSourceCanonicalOwner(javaType string, def *symbol.Definition, ctx Ctx) string {
	element, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	base, args := parseJavaTypeString(element)
	first := strings.Split(base, ".")[0]
	if rank != 0 || len(args) != 0 || base == "" || visibleTypeParameterDeclarationForJavaType(first, ctx) != nil || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil || resolveClassScopeByQualifiedName(ctx, first) != nil {
		return ""
	}
	if def != nil && (def.DirectTypeParameter != nil || def.TypeParameterBindings[first] != nil || def.TypeParameterBindings[base] != nil) {
		return ""
	}
	// Catalog ownership is proved by the actual retained source declaration.
	// Synthetic declarations belong to that file only through Node.Equal ancestry.
	proof := ctx.Clone()
	if proof.currentClass == nil || proof.currentClass.Class == nil || proof.currentFile == nil {
		return ""
	}
	if findFileScopeForClassScope(proof.currentClass, proof) != proof.currentFile {
		actual := proof.currentClass
		owned := false
		for parent := actual.Enclosing; parent != nil; parent = parent.Enclosing {
			if parent.Class != nil && threadSourceNodeWithin(actual.Class.DeclarationNode, parent.Class.DeclarationNode) && findFileScopeForClassScope(parent, proof) == proof.currentFile {
				proof.currentClass = parent
				owned = true
				break
			}
		}
		if !owned {
			return ""
		}
	}
	bindings := symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))
	if def != nil {
		if bindings == nil && len(def.TypeParameterBindings) > 0 {
			bindings = make(map[string]*symbol.TypeParamDeclaration, len(def.TypeParameterBindings))
		}
		for name, parameter := range def.TypeParameterBindings {
			bindings[name] = parameter
		}
	}
	owner := declaredExternalNominalOwner(base, bindings, proof)
	if owner == "java.lang.Thread" || owner == "java.lang.Runnable" {
		return owner
	}
	return ""
}
func threadSourceDirectRoot(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil || scope.Class == nil || scope.IsInterface || scope.IsEnum {
		return false
	}
	declaring, ok := threadSourceDeclarationCtx(scope, nil, ctx)
	return ok && threadSourceCanonicalOwner(scope.Superclass, nil, declaring) == "java.lang.Thread"
}
func threadSourceRoot(scope *symbol.ClassScope, ctx Ctx) *symbol.ClassScope {
	seen := map[*symbol.ClassScope]bool{}
	for scope != nil && !seen[scope] {
		seen[scope] = true
		if threadSourceDirectRoot(scope, ctx) {
			return scope
		}
		if _, ok := threadSourceDeclarationCtx(scope, nil, ctx); !ok {
			return nil
		}
		scope = resolveSuperclassScopeInDeclaringContext(ctx, scope)
	}
	return nil
}
func threadSourceOverrideSlot(method *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) string {
	if method == nil || method.Constructor || method.IsStatic || method.IsPrivate || len(method.Parameters) != 0 || len(method.TypeParameters) != 0 || method.DeclarationNode == nil || !declarationHasModifier(method.DeclarationNode, "public") || threadSourceRoot(owner, ctx) == nil {
		return ""
	}
	result := strings.TrimSpace(method.OriginalType)
	switch method.OriginalName {
	case "interrupt":
		if result == "void" {
			return "Interrupt"
		}
	case "isInterrupted":
		if result == "boolean" {
			return "IsInterrupted"
		}
	case "run":
		if result == "void" {
			return "Run"
		}
	case "start":
		if result == "void" {
			return "Start"
		}
	}
	return ""
}
func threadSourceSelectedOverride(scope *symbol.ClassScope, name string, ctx Ctx) (*symbol.Definition, *symbol.ClassScope) {
	seen := map[*symbol.ClassScope]bool{}
	for scope != nil && !seen[scope] {
		seen[scope] = true
		for _, m := range scope.Methods {
			if m != nil && m.OriginalName == name && threadSourceOverrideSlot(m, scope, ctx) != "" {
				return m, scope
			}
		}
		scope = resolveSuperclassScopeInDeclaringContext(ctx, scope)
	}
	return nil, nil
}
