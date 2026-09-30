package nodeutil

import sitter "github.com/smacker/go-tree-sitter"

// NamedChildrenOf gets all named children of a given node
func NamedChildrenOf(node *sitter.Node) []*sitter.Node {
	count := int(node.NamedChildCount())
	children := make([]*sitter.Node, count)
	for i := 0; i < count; i++ {
		children[i] = node.NamedChild(i)
	}
	return children
}

// UnnamedChildrenOf gets all the named + unnamed children of a given node
func UnnamedChildrenOf(node *sitter.Node) []*sitter.Node {
	count := int(node.ChildCount())
	children := make([]*sitter.Node, count)
	for i := 0; i < count; i++ {
		children[i] = node.Child(i)
	}
	return children
}

// VariableDeclarators returns every declarator carried by a Java declaration
// in source order.
func VariableDeclarators(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == "variable_declarator" {
		return []*sitter.Node{node}
	}
	var declarators []*sitter.Node
	for _, child := range NamedChildrenOf(node) {
		if child.Type() == "variable_declarator" {
			declarators = append(declarators, child)
		}
	}
	return declarators
}

// JavaParameterNodes returns the declared type and name of a formal or varargs
// parameter. Varargs have a variable_declarator child; modifiers and annotations
// may precede its type, so positional child indices are not stable.
func JavaParameterNodes(node *sitter.Node) (typeNode, nameNode *sitter.Node) {
	if node == nil {
		return nil, nil
	}
	if node.Type() != "spread_parameter" {
		return node.ChildByFieldName("type"), node.ChildByFieldName("name")
	}
	for _, child := range NamedChildrenOf(node) {
		switch child.Type() {
		case "modifiers", "annotation", "marker_annotation", "line_comment", "block_comment":
			continue
		case "variable_declarator":
			nameNode = child.ChildByFieldName("name")
		default:
			if typeNode == nil {
				typeNode = child
			}
		}
	}
	return typeNode, nameNode
}
