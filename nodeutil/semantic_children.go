package nodeutil

import sitter "github.com/smacker/go-tree-sitter"

// semanticJavaChild excludes only the Java grammar's comment extras. Unknown
// syntax and ERROR nodes stay visible so strict conversion still diagnoses them.
func semanticJavaChild(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	if node.IsExtra() {
		switch node.Type() {
		case "line_comment", "block_comment":
			return false
		}
	}
	return true
}

// SemanticNamedChildrenOf returns the syntax roles in source order. Tree-sitter
// includes comment extras among named children; expression and argument parsing
// must not count those as operands, parameters, or values. Raw AST traversal and
// documentation collection continue to use NamedChildrenOf.
func SemanticNamedChildrenOf(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}
	children := make([]*sitter.Node, 0, node.NamedChildCount())
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if semanticJavaChild(child) {
			children = append(children, child)
		}
	}
	return children
}

func SemanticNamedChildCount(node *sitter.Node) int {
	if node == nil {
		return 0
	}
	count := 0
	for i := 0; i < int(node.NamedChildCount()); i++ {
		if semanticJavaChild(node.NamedChild(i)) {
			count++
		}
	}
	return count
}

func SemanticNamedChild(node *sitter.Node, index int) *sitter.Node {
	if node == nil || index < 0 {
		return nil
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if !semanticJavaChild(child) {
			continue
		}
		if index == 0 {
			return child
		}
		index--
	}
	return nil
}
