package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Only a declaration-owned tri-state proof can refuse an invariant argument.
func declaredInvariantArgumentDisagreement(actual, formal inferredJavaTypeOrigin, ctx Ctx) bool {
	if !declaredProofOwnerHeader(actual, ctx) || !declaredProofOwnerHeader(formal, ctx) || actual.nominalScope != formal.nominalScope || actual.externalName != formal.externalName || len(actual.arguments) != len(formal.arguments) {
		return false
	}
	// Parent projection preserves each supplied argument origin and its syntax,
	// while its own Java spelling can contain only the canonical parent head.
	// No wildcard syntax is reconstructed from that head or an expected type.
	for i, argument := range actual.arguments {
		other := formal.arguments[i]
		if !argument.wildcard && !other.wildcard && declaredInvariantArgumentEquivalence(argument, other, ctx) == declaredArgumentDifferent {
			return true
		}
	}
	return false
}

// Only a genuine unbounded caller declaration can prove this refusal. This is
// not an erasure of explicit bounds and cannot grant a new applicable method.
func declaredUnboundedCallerParameterRefusal(actual, formal inferredJavaTypeOrigin, ctx Ctx) bool {
	if actual.parameter == nil || actual.arrayRank != 0 || actual.wildcard || actual.unresolved || formal.parameter != nil || formal.arrayRank != 0 || formal.externalName == "java.lang.Object" || formal.nominalScope == nil && formal.externalName == "" || !closedDeclaredAncestryOrigin(formal) {
		return false
	}
	// Explicitly bounded declarations are outside this refusal. This also avoids
	// building a source certificate for the common bounded positive case.
	if ctx.localScope != nil {
		for _, p := range ctx.localScope.TypeParameters {
			if p.Declaration == actual.parameter && len(p.Bounds) > 0 {
				return false
			}
		}
	}
	if ctx.currentClass != nil {
		for _, p := range ctx.currentClass.TypeParameters {
			if p.Declaration == actual.parameter && len(p.Bounds) > 0 {
				return false
			}
		}
	}
	owners := declaredRefusalLexicalOwners(ctx)
	if len(owners) == 0 {
		return false
	}
	var declaration *sitter.Node
	found := false
	accept := func(node *sitter.Node, p symbol.TypeParam) bool {
		if p.Declaration != actual.parameter {
			return true
		}
		if node == nil || len(p.Bounds) != 0 {
			return false
		}
		if found && !declaration.Equal(node) {
			return false
		}
		declaration = node
		found = true
		return true
	}
	for _, owner := range owners {
		if ctx.localScope == nil || !ctx.localScope.IsStatic {
			if classScopeCarriesTypeParameterDeclaration(ctx.currentClass, actual.parameter) {
				for _, p := range owner.OwnTypeParameters() {
					if !accept(owner.Class.DeclarationNode, p) {
						return false
					}
				}
			}
		}
		for _, method := range owner.Methods {
			if method == nil || method.DeclarationNode == nil || !threadSourceNodeWithin(method.DeclarationNode, owner.Class.DeclarationNode) {
				continue
			}
			direct := ctx.localScope != nil && ctx.localScope.DeclarationNode != nil && ctx.localScope.DeclarationNode.Equal(method.DeclarationNode)
			captured := classScopeCarriesTypeParameterDeclaration(ctx.currentClass, actual.parameter) && threadSourceNodeWithin(ctx.currentClass.Class.DeclarationNode, method.DeclarationNode)
			if !direct && !captured {
				continue
			}
			for _, p := range method.TypeParameters {
				if !accept(method.DeclarationNode, p) {
					return false
				}
			}
		}
	}
	return found
}

// Validate the complete actual lexical chain before trusting any caller slot.
// Neither a matching name nor the current file is a registration certificate.
func declaredRefusalLexicalOwners(ctx Ctx) []*symbol.ClassScope {
	seen := map[*symbol.ClassScope]bool{}
	var owners []*symbol.ClassScope
	var file *symbol.FileScope
	for current := ctx.currentClass; current != nil; current = current.Enclosing {
		if seen[current] || current.Class == nil || current.Class.DeclarationNode == nil {
			return nil
		}
		seen[current] = true
		actualFile := declaredRefusalRegisteredScopeFile(current, ctx)
		if actualFile == nil || file != nil && file != actualFile {
			return nil
		}
		file = actualFile
		if len(owners) > 0 {
			child := owners[len(owners)-1].Class.DeclarationNode
			if child.Equal(current.Class.DeclarationNode) || !threadSourceNodeWithin(child, current.Class.DeclarationNode) {
				return nil
			}
		}
		owners = append(owners, current)
	}
	if ctx.localScope != nil && (ctx.localScope.DeclarationNode == nil || len(owners) == 0 || !threadSourceNodeWithin(ctx.localScope.DeclarationNode, owners[0].Class.DeclarationNode)) {
		return nil
	}
	return owners
}

func declaredRefusalRegisteredScopeFile(scope *symbol.ClassScope, ctx Ctx) *symbol.FileScope {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil {
		return nil
	}
	if file := declaredRefusalOrdinaryScopeFile(scope); file != nil {
		return file
	}
	// Late original declarations use the existing live-registry/lexical proof,
	// after deriving an actual registered ancestor file independently of ctx.
	seen := map[*symbol.ClassScope]bool{scope: true}
	child := scope
	for owner := scope.Enclosing; owner != nil; owner = owner.Enclosing {
		if seen[owner] || owner.Class == nil || owner.Class.DeclarationNode == nil || child.Class.DeclarationNode.Equal(owner.Class.DeclarationNode) || !threadSourceNodeWithin(child.Class.DeclarationNode, owner.Class.DeclarationNode) {
			return nil
		}
		seen[owner] = true
		if file := declaredRefusalOrdinaryScopeFile(owner); file != nil && intrinsicLexicalDeclaringFile(scope, file, ctx) {
			return file
		}
		child = owner
	}
	return nil
}

func declaredRefusalOrdinaryScopeFile(want *symbol.ClassScope) *symbol.FileScope {
	if symbol.GlobalScope == nil {
		return nil
	}
	var result *symbol.FileScope
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil || file.ParsedRoot() == nil {
				continue
			}
			seen := map[*symbol.ClassScope]bool{}
			var visit func(*symbol.ClassScope) bool
			visit = func(scope *symbol.ClassScope) bool {
				if scope == nil || seen[scope] {
					return false
				}
				seen[scope] = true
				if scope == want {
					return scope.Class != nil && scope.Class.DeclarationNode != nil && threadSourceNodeWithin(scope.Class.DeclarationNode, file.ParsedRoot())
				}
				for _, child := range scope.Subclasses {
					if visit(child) {
						return true
					}
				}
				return false
			}
			for _, top := range file.TopLevelClasses {
				if visit(top) {
					if result != nil && result != file {
						return nil
					}
					result = file
					break
				}
			}
		}
	}
	return result
}
