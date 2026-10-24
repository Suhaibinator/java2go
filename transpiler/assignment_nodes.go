package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

// Assignment operands are grammar fields; comment extras are not operands or
// punctuation. Reading roles never emits an operand, so each lowering retains
// its existing address/load/RHS/store sequencing and single evaluation.
func assignmentExpressionNodes(node *sitter.Node, source []byte) (left, operator, right *sitter.Node, valid bool) {
	if node == nil || node.Type() != "assignment_expression" || node.IsMissing() || node.HasError() {
		return nil, nil, nil, false
	}
	left, right = node.ChildByFieldName("left"), node.ChildByFieldName("right")
	if left == nil || right == nil || left.IsMissing() || right.IsMissing() || left.HasError() || right.HasError() {
		return nil, nil, nil, false
	}
	operatorField := node.ChildByFieldName("operator")
	for _, child := range nodeutil.SemanticNamedChildrenOf(node) {
		if assignmentSameRoleNode(child, left) || assignmentSameRoleNode(child, right) || (operatorField != nil && assignmentSameRoleNode(child, operatorField)) {
			continue
		}
		// The shared helper excludes only known Java comment extras. Unknown syntax
		// and ERROR remain visible and must never be silently selected as a role.
		return nil, nil, nil, false
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.IsNamed() {
			continue
		}
		if child.IsExtra() && (child.Type() == "line_comment" || child.Type() == "block_comment") {
			continue
		}
		if !assignmentOperatorNode(child, left, right, source) || operator != nil || (operatorField != nil && !assignmentSameRoleNode(child, operatorField)) {
			return nil, nil, nil, false
		}
		operator = child
	}
	if operator == nil && assignmentOperatorNode(operatorField, left, right, source) {
		operator = operatorField
	}
	if operator == nil {
		return nil, nil, nil, false
	}
	return left, operator, right, true
}

func assignmentSameRoleNode(a, b *sitter.Node) bool {
	return a != nil && b != nil && a.Type() == b.Type() && a.StartByte() == b.StartByte() && a.EndByte() == b.EndByte()
}

func assignmentOperatorNode(node, left, right *sitter.Node, source []byte) bool {
	if node == nil || node.IsMissing() || node.HasError() || node.IsExtra() || node.StartByte() < left.EndByte() || node.EndByte() > right.StartByte() {
		return false
	}
	switch node.Content(source) {
	case "=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", ">>>=":
		return true
	default:
		return false
	}
}
