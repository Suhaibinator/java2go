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
