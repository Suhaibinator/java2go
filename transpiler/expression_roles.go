package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

// javaUpdateExpressionParts keeps the operand and grammar token separate from
// syntactic comment extras. Byte order distinguishes prefix from postfix.
func javaUpdateExpressionParts(node *sitter.Node) (operand, operator *sitter.Node, post bool) {
	operand = nodeutil.SemanticNamedChild(node, 0)
	for index := 0; index < int(node.ChildCount()); index++ {
		child := node.Child(index)
		if child.Type() == "++" || child.Type() == "--" {
			operator = child
			break
		}
	}
	if operand != nil && operator != nil {
		post = operand.StartByte() < operator.StartByte()
	}
	return
}
