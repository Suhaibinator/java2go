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
	if simpleReceiverHasLexicalValueBinding(ctx, source, node, root) {
		return nil
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
