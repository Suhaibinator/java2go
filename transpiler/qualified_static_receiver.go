package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func qualifiedSourceClassReceiver(ctx Ctx, source []byte, node *sitter.Node) *symbol.ClassScope {
	name, root := qualifiedReceiverName(node, source)
	if name == "" || root == "" || !strings.Contains(name, ".") {
		return nil
	}
	if ctx.localScope != nil && ctx.localScope.ParameterByName(root) != nil {
		return nil
	}
	// Fields declared by enclosing lexical classes share the value namespace
	// with fields of the current class. Inspect that declaration chain directly:
	// whole-method local inference also sees declarations after this expression.
	seen := map[*symbol.ClassScope]struct{}{}
	for scope := ctx.currentClass; scope != nil; scope = scope.Enclosing {
		if _, duplicate := seen[scope]; duplicate {
			return nil
		}
		seen[scope] = struct{}{}
		if findFieldResolutionInHierarchy(scope, root, ctx) != nil {
			return nil
		}
	}
	// Local symbols are collected for a whole Java method. Inspect lexical
	// scopes here so a later declaration cannot hide an earlier package name.
	for child, parent := node, node.Parent(); parent != nil; child, parent = parent, parent.Parent() {
		switch parent.Type() {
		case "block", "constructor_body":
			for _, sibling := range nodeutil.NamedChildrenOf(parent) {
				if sibling.StartByte() >= child.StartByte() {
					break
				}
				if localDeclarationNames(sibling, root, source) {
					return nil
				}
			}
		case "for_statement":
			if localDeclarationNames(parent.ChildByFieldName("init"), root, source) {
				return nil
			}
		case "enhanced_for_statement":
			body := parent.ChildByFieldName("body")
			variable := parent.ChildByFieldName("name")
			if body != nil && variable != nil && node.StartByte() >= body.StartByte() && variable.Content(source) == root {
				return nil
			}
		}
	}
	return resolveClassScopeByQualifiedName(ctx, name)
}

func qualifiedReceiverName(node *sitter.Node, source []byte) (string, string) {
	if node == nil {
		return "", ""
	}
	if node.Type() == "identifier" {
		name := node.Content(source)
		return name, name
	}
	if node.Type() != "field_access" {
		return "", ""
	}
	prefix, root := qualifiedReceiverName(node.ChildByFieldName("object"), source)
	field := node.ChildByFieldName("field")
	if prefix == "" || field == nil || field.Type() != "identifier" {
		return "", ""
	}
	return prefix + "." + field.Content(source), root
}

func localDeclarationNames(node *sitter.Node, name string, source []byte) bool {
	if node == nil || node.Type() != "local_variable_declaration" {
		return false
	}
	for _, child := range nodeutil.NamedChildrenOf(node) {
		if child.Type() == "variable_declarator" {
			if ident := child.ChildByFieldName("name"); ident != nil && ident.Content(source) == name {
				return true
			}
		}
	}
	return false
}
