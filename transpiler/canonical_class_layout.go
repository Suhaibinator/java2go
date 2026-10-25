package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Class arguments describe the represented Java type, not the allocation of
// the Class object. The runtime stores every such object as the same *Class;
// Class arrays keep their existing nominal Class component and exact rank.
// This proof changes storage eligibility only, never the logical type tree.
func canonicalClassArgumentIndependent(javaType string, ctx Ctx, occurrences ...*sitter.Node) bool {
	component, _ := javaArrayTypeParts(strings.TrimSpace(javaType))
	base, arguments, valid := reflectiveNamedTypeParts(component)
	if !valid || stripJavaQualifier(base) != "Class" || len(arguments) > 1 {
		return false
	}
	owner, known := canonicalArgumentIndependentOccurrenceOwner(base, ctx, occurrences...)
	return known && owner == "java.lang.Class"
}

// Argument-independent layouts still require the occurrence's original Java
// namespace. A logical type name alone cannot establish a native declaration.
func canonicalArgumentIndependentOccurrenceOwner(base string, ctx Ctx, occurrences ...*sitter.Node) (string, bool) {
	if len(occurrences) != 1 || occurrences[0] == nil || ctx.currentFile == nil {
		return "", false
	}
	occurrence := occurrences[0]
	source := ctx.currentFile.Source
	if occurrence.EndByte() > uint32(len(source)) || occurrence.StartByte() >= occurrence.EndByte() {
		return "", false
	}
	prefix := strings.SplitN(base, ".", 2)[0]
	if visibleTypeParameterDeclarationForJavaType(prefix, ctx) != nil || sourceGenericViewLocalTypeShadowed(occurrence, base, source) {
		return "", false
	}
	// A method-wide hoisted registry cannot establish visibility at an earlier
	// occurrence. Its original lexical declarations were checked above instead.
	lookup := ctx.Clone()
	lookup.localClasses = nil
	if owner := lookup.currentClass; owner != nil && owner.Class != nil && owner.Class.DeclarationNode != nil {
		declaration := owner.Class.DeclarationNode
		if receiverNodeWithin(occurrence, declaration) && !receiverNodeWithin(occurrence, declaration.ChildByFieldName("body")) {
			lookup = classHeaderTypeCtx(owner, lookup)
		}
	}
	if classServiceASTTypePrefix(occurrence, prefix, source, lookup) || classServiceSourceTypePrefix(prefix, lookup) {
		return "", false
	}
	if resolveClassScopeByQualifiedName(lookup, base) != nil {
		return "", false
	}
	if strings.Contains(base, ".") {
		// A single-type import binds this first TYPE segment even if its source
		// declaration is outside the loaded graph; a failed member lookup is final.
		if _, imported := lookup.currentFile.Imports[prefix]; imported {
			return "", false
		}
	}
	if classServiceUnknownImportPrefix(prefix, lookup) {
		return "", false
	}
	owner, known := canonicalIntrinsicOwner(base, lookup)
	return owner, known && intrinsicOwnerSupported(owner)
}

// The occurrence's actual ancestors supply unhoisted member namespaces. Member
// types cover their entire body, unlike ordered local types. A body's members
// do not enter its own declaration header, except for record headers.
func classServiceASTTypePrefix(occurrence *sitter.Node, name string, source []byte, ctx Ctx) bool {
	for ancestor := occurrence.Parent(); ancestor != nil; ancestor = ancestor.Parent() {
		var body *sitter.Node
		switch ancestor.Type() {
		case "class_body", "interface_body", "enum_body", "annotation_type_body":
			body = ancestor
		case "record_declaration":
			body = ancestor.ChildByFieldName("body")
		}
		if body == nil {
			continue
		}
		for _, declaration := range nodeutil.NamedChildrenOf(body) {
			declarations := []*sitter.Node{declaration}
			if declaration.Type() == "enum_body_declarations" {
				declarations = nodeutil.NamedChildrenOf(declaration)
			}
			for _, member := range declarations {
				if sourceGenericViewTypeDeclaration(member) {
					ident := member.ChildByFieldName("name")
					if ident != nil && ident.Content(source) == name {
						return true
					}
				}
			}
		}
		if creation := body.Parent(); creation != nil && creation.Type() == "object_creation_expression" {
			typ := creation.ChildByFieldName("type")
			if typ == nil {
				return true
			}
			base, _ := parseJavaTypeString(typ.Content(source))
			parent := resolveClassScopeByQualifiedName(ctx, base)
			// An uninventoried anonymous superclass could introduce a member type.
			if parent == nil || classServiceHierarchyTypePrefix(parent, name, ctx, map[*symbol.ClassScope]bool{}) {
				return true
			}
		}
	}
	return false
}

func classServiceSourceTypePrefix(name string, ctx Ctx) bool {
	if resolveClassScopeByQualifiedName(ctx, name) != nil {
		return true
	}
	for scope := ctx.currentClass; scope != nil; scope = scope.Enclosing {
		if scope == ctx.memberTypeHeaderOwner && !classHeaderIncludesMemberTypes(scope) {
			continue
		}
		if classServiceHierarchyTypePrefix(scope, name, ctx, map[*symbol.ClassScope]bool{}) {
			return true
		}
	}
	return false
}

// Nil member resolution can also mean ambiguity. Preserve every accessible
// source member candidate; unresolved declared parents cannot prove absence.
func classServiceHierarchyTypePrefix(scope *symbol.ClassScope, name string, ctx Ctx, path map[*symbol.ClassScope]bool) bool {
	candidates, unknown := classServiceHierarchyTypeCandidates(scope, name, ctx, path)
	return unknown || len(candidates) != 0
}

func classServiceHierarchyTypeCandidates(scope *symbol.ClassScope, name string, ctx Ctx, path map[*symbol.ClassScope]bool) ([]*symbol.ClassScope, bool) {
	if scope == nil || path[scope] {
		return nil, true
	}
	path[scope] = true
	defer delete(path, scope)
	var candidates []*symbol.ClassScope
	for _, member := range scope.Subclasses {
		if member.Class != nil && member.Class.OriginalName == name {
			candidates = append(candidates, member)
		}
	}
	if len(candidates) != 0 {
		return candidates, false
	}
	unknown := false
	declaration := classHeaderTypeCtx(scope, ctx)
	for _, typ := range append([]string{scope.Superclass}, scope.ImplementedInterfaces...) {
		if typ == "" {
			continue
		}
		base, _ := parseJavaTypeString(typ)
		parent := resolveClassScopeByQualifiedName(declaration, base)
		if parent == nil {
			unknown = true
			continue
		}
		inherited, unresolved := classServiceHierarchyTypeCandidates(parent, name, declaration, path)
		unknown = unknown || unresolved
		for _, member := range inherited {
			if memberTypeInheritedBy(member, scope) {
				candidates = append(candidates, member)
			}
		}
	}
	return candidates, unknown
}

func classServiceUnknownImportPrefix(name string, ctx Ctx) bool {
	for _, imported := range staticMethodImports(ctx) {
		if !imported.wildcard && imported.member != name {
			continue
		}
		owner := findQualifiedSourceClass(imported.owner)
		if owner == nil {
			return true
		}
		if member, present := staticImportedSourceType(name, imported.wildcard, ctx); present || member != nil {
			return true
		}
	}
	if _, explicit := ctx.currentFile.Imports[name]; explicit {
		return false
	}
	for _, pkg := range intrinsicOnDemandImports(ctx) {
		if pkg != "java.lang" && symbol.GlobalScope.FindPackage(pkg) == nil {
			return true
		}
	}
	return false
}

// Select a source type occurrence, not a declaration's start position. The
// same immutable clause nodes populated Superclass/ImplementedInterfaces.
func canonicalClassFamilyTypeOccurrence(javaType string, node *sitter.Node, source []byte) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "object_creation_expression" {
		typ := node.ChildByFieldName("type")
		if typ != nil && typ.Content(source) == javaType {
			return typ
		}
		return nil
	}
	var selected *sitter.Node
	for _, field := range []string{"superclass", "interfaces", "extends_interfaces"} {
		clause := node.ChildByFieldName(field)
		if clause == nil {
			for _, child := range nodeutil.NamedChildrenOf(node) {
				if child.Type() == field {
					clause = child
					break
				}
			}
		}
		for _, typ := range collectTypeNodes(clause) {
			if typ.Content(source) == javaType {
				if selected != nil && !sameSourceNode(selected, typ) {
					return nil
				}
				selected = typ
			}
		}
	}
	return selected
}

func canonicalClassFamilyHeaderTypeOccurrence(javaType string, node *sitter.Node, ctx Ctx) *sitter.Node {
	if ctx.currentFile == nil {
		return nil
	}
	return canonicalClassFamilyTypeOccurrence(javaType, node, ctx.currentFile.Source)
}
