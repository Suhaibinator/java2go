package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"

	sitter "github.com/smacker/go-tree-sitter"
)

// Member types are resolved in the nearest enclosing declaration, not by a
// recursive search through unrelated member declarations in source-file order.
func lexicalMemberType(name string, ctx Ctx) *symbol.ClassScope {
	parts := strings.Split(name, ".")
	for owner := ctx.currentClass; owner != nil; owner = owner.Enclosing {
		if owner.Class != nil && owner.Class.OriginalName == parts[0] {
			if found := directMemberTypePath(owner, parts[1:], ctx); found != nil {
				return found
			}
		}
		if owner == ctx.memberTypeHeaderOwner && !classHeaderIncludesMemberTypes(owner) {
			continue
		}
		if member := memberTypeInHierarchy(owner, parts[0], ctx); member != nil {
			return directMemberTypePath(member, parts[1:], ctx)
		}
	}
	return nil
}

// A member declared in the nearest class hides inherited members. Otherwise
// lookup follows only that class's declared supertypes, never sibling bodies.
func memberTypeInHierarchy(owner *symbol.ClassScope, name string, ctx Ctx) *symbol.ClassScope {
	if owner == nil {
		return nil
	}
	for _, child := range owner.Subclasses {
		if child.Class != nil && child.Class.OriginalName == name {
			return child
		}
	}
	if ctx.memberTypeLookupPath[owner] || (owner.Superclass == "" && len(owner.ImplementedInterfaces) == 0) {
		return nil
	}
	// Resolving an extends clause itself performs lexical lookup. A fresh path
	// prevents that lookup from rediscovering the hierarchy currently being
	// resolved, without mutating the caller or poisoning subsequent lookups.
	lookup := ctx.Clone()
	lookup.memberTypeLookupPath = make(map[*symbol.ClassScope]bool, len(ctx.memberTypeLookupPath)+1)
	for active := range ctx.memberTypeLookupPath {
		lookup.memberTypeLookupPath[active] = true
	}
	lookup.memberTypeLookupPath[owner] = true
	parents := []*symbol.ClassScope{resolveSuperclassScopeInDeclaringContext(lookup, owner)}
	parents = append(parents, resolveImplementedInterfaceScopesInDeclaringContext(lookup, owner)...)
	var found *symbol.ClassScope
	for _, parent := range parents {
		candidate := memberTypeInHierarchy(parent, name, lookup)
		if !memberTypeInheritedBy(candidate, owner) {
			continue
		}
		if found != nil && found != candidate {
			// Distinct inherited member declarations are ambiguous. A shared
			// declaration reached through a diamond remains the same member.
			return nil
		}
		found = candidate
	}
	return found
}

// Accessibility is checked at every inheritance edge: a package member lost
// across a package boundary does not reappear in a later subclass that happens
// to return to the declaring package. Interface member types are implicitly
// public; class modifiers come from the declaration, not method-only flags.
func memberTypeInheritedBy(member, inheritor *symbol.ClassScope) bool {
	if member == nil || member.Class == nil {
		return false
	}
	declaration := member.Class.DeclarationNode
	if member.Class.IsPrivate || declarationHasModifier(declaration, "private") {
		return false
	}
	if (member.Enclosing != nil && member.Enclosing.IsInterface) ||
		declarationHasModifier(declaration, "public") || declarationHasModifier(declaration, "protected") {
		return true
	}
	memberPackage, memberKnown := lexicalMemberPackage(member)
	inheritorPackage, inheritorKnown := lexicalMemberPackage(inheritor)
	return memberKnown && inheritorKnown && memberPackage == inheritorPackage
}

// Hoisted local and anonymous classes are not entries in the named source
// inventory. Their lexical owner still establishes their Java package; an
// unknown scope must not be mistaken for a class in the unnamed package.
func lexicalMemberPackage(scope *symbol.ClassScope) (string, bool) {
	for owner := scope; owner != nil; owner = owner.Enclosing {
		if file := findFileScopeForClassScope(owner); file != nil {
			return file.Package, true
		}
	}
	return "", false
}

func directMemberTypePath(owner *symbol.ClassScope, members []string, ctx Ctx) *symbol.ClassScope {
	for _, name := range members {
		owner = memberTypeInHierarchy(owner, name, ctx)
		if owner == nil {
			return nil
		}
	}
	return owner
}

func declaredFileType(name string, file *symbol.FileScope, ctx Ctx) *symbol.ClassScope {
	if file == nil {
		return nil
	}
	parts := strings.Split(name, ".")
	for _, top := range file.TopLevelClasses {
		if top.Class != nil && top.Class.OriginalName == parts[0] {
			return directMemberTypePath(top, parts[1:], ctx)
		}
	}
	return nil
}

// Relative member paths retain their first type binding across compilation
// units: an imported or same-package Outer owns Outer.Inner. Resolve the root
// normally, then walk only that declaration's member hierarchy.
func relativeMemberType(name string, ctx Ctx) *symbol.ClassScope {
	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return nil
	}
	owner := resolveClassScopeByQualifiedName(ctx, parts[0])
	if owner == nil {
		return nil
	}
	return directMemberTypePath(owner, parts[1:], ctx)
}

// Class and interface members begin at the body; record members also include
// the header. Preserve the original declaration pointer and binder identities
// while selecting that lookup region, rather than synthesizing a class scope.
func classHeaderTypeCtx(scope *symbol.ClassScope, ctx Ctx) Ctx {
	result := classScopeCtx(scope, ctx)
	result.memberTypeHeaderOwner = scope
	return result
}

func classHeaderIncludesMemberTypes(scope *symbol.ClassScope) bool {
	return scope != nil && scope.Class != nil && scope.Class.DeclarationNode != nil &&
		scope.Class.DeclarationNode.Type() == "record_declaration"
}

// Select context from the actual declaration child, not from its spelling:
// identical p.Cell text can denote different declarations in header and body.
func classDeclarationChildCtx(root, child *sitter.Node, ctx Ctx) Ctx {
	if !sourceGenericViewTypeDeclaration(root) || ctx.currentClass == nil ||
		ctx.currentClass.Class == nil || !sameSourceNode(root, ctx.currentClass.Class.DeclarationNode) {
		return ctx
	}
	switch child.Type() {
	case "class_body", "interface_body", "enum_body", "annotation_type_body":
		return classScopeCtx(ctx.currentClass, ctx)
	}
	return classHeaderTypeCtx(ctx.currentClass, ctx)
}
