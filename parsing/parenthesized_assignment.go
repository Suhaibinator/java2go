package parsing

import (
	"bytes"
	sitter "github.com/smacker/go-tree-sitter"
)

type parenthesizedAssignmentRepair struct {
	leftStart, leftEnd, operatorStart, operatorEnd uint32
	parentheses                                    []uint32
}

// The bundled grammar excludes parenthesized variable references from its
// assignment left production. Recover only AST-identified variable references,
// using a private equal-length parser buffer. Input bytes and every source
// offset stay unchanged. Literal text and non-variable expressions are excluded.
func parenthesizedAssignmentParserSource(root *sitter.Node, source []byte) ([]byte, []parenthesizedAssignmentRepair) {
	if root == nil || !root.HasError() {
		return nil, nil
	}
	var repairs []parenthesizedAssignmentRepair
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node.Type() == "parenthesized_expression" && !node.HasError() {
			if repair, ok := parenthesizedAssignmentVariable(node, source); ok {
				repairs = append(repairs, repair)
			}
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			visit(node.NamedChild(i))
		}
	}
	visit(root)
	if len(repairs) == 0 {
		return nil, nil
	}
	parserSource := append([]byte(nil), source...)
	for _, repair := range repairs {
		for _, offset := range repair.parentheses {
			parserSource[offset] = ' '
		}
	}
	return parserSource, repairs
}

func parenthesizedAssignmentVariable(node *sitter.Node, source []byte) (parenthesizedAssignmentRepair, bool) {
	repair := parenthesizedAssignmentRepair{}
	operatorStart, operatorEnd, ok := assignmentTokenAfterNode(node.EndByte(), source)
	if !ok {
		return repair, false
	}
	repair.operatorStart, repair.operatorEnd = operatorStart, operatorEnd
	for node.Type() == "parenthesized_expression" {
		if node.HasError() || node.ChildCount() < 3 {
			return parenthesizedAssignmentRepair{}, false
		}
		open, close := node.Child(0), node.Child(int(node.ChildCount())-1)
		if open.Type() != "(" || close.Type() != ")" || open.IsMissing() || close.IsMissing() || open.EndByte() != open.StartByte()+1 || close.EndByte() != close.StartByte()+1 {
			return parenthesizedAssignmentRepair{}, false
		}
		if source[open.StartByte()] != '(' || source[close.StartByte()] != ')' {
			return parenthesizedAssignmentRepair{}, false
		}
		repair.parentheses = append(repair.parentheses, open.StartByte(), close.StartByte())
		var expression *sitter.Node
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)
			if child.Type() == "block_comment" || child.Type() == "line_comment" || child.Type() == "comment" {
				continue
			}
			if expression != nil {
				return parenthesizedAssignmentRepair{}, false
			}
			expression = child
		}
		if expression == nil {
			return parenthesizedAssignmentRepair{}, false
		}
		node = expression
	}
	if node.HasError() || node.IsMissing() {
		return parenthesizedAssignmentRepair{}, false
	}
	switch node.Type() {
	case "identifier", "field_access", "array_access":
	default:
		return parenthesizedAssignmentRepair{}, false
	}
	repair.leftStart, repair.leftEnd = node.StartByte(), node.EndByte()
	return repair, true
}

// Read a Java assignment token after an AST node, allowing Java trivia. This
// does not scan for names or rewrite arbitrary substrings; the accepted token
// and both variable boundaries must be confirmed by the reparsed grammar.
func assignmentTokenAfterNode(offset uint32, source []byte) (uint32, uint32, bool) {
	i := int(offset)
	for i < len(source) {
		switch source[i] {
		case ' ', '\t', '\r', '\n', '\f':
			i++
			continue
		}
		if i+1 < len(source) && source[i] == '/' && source[i+1] == '/' {
			i += 2
			for i < len(source) && source[i] != '\n' && source[i] != '\r' {
				i++
			}
			continue
		}
		if i+1 < len(source) && source[i] == '/' && source[i+1] == '*' {
			end := bytes.Index(source[i+2:], []byte("*/"))
			if end < 0 {
				return 0, 0, false
			}
			i += end + 4
			continue
		}
		break
	}
	for _, operator := range []string{">>>=", ">>=", "<<=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "="} {
		if i+len(operator) > len(source) || string(source[i:i+len(operator)]) != operator {
			continue
		}
		if operator == "=" && i+1 < len(source) && (source[i+1] == '=' || source[i+1] == '>') {
			return 0, 0, false
		}
		return uint32(i), uint32(i + len(operator)), true
	}
	return 0, 0, false
}

// A repair is admitted only when the grammar recovers a complete assignment
// with the exact original variable/operator spans. Unsupported or malformed
// targets keep the original ERROR tree rather than being treated as Java.
func parenthesizedAssignmentsRecovered(root *sitter.Node, source []byte, repairs []parenthesizedAssignmentRepair) bool {
	found := make([]bool, len(repairs))
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node.Type() == "assignment_expression" && !node.HasError() && !node.IsMissing() {
			left, right := node.ChildByFieldName("left"), node.ChildByFieldName("right")
			if left != nil && right != nil && !left.IsMissing() && !right.IsMissing() {
				for i, repair := range repairs {
					if left.StartByte() != repair.leftStart || left.EndByte() != repair.leftEnd {
						continue
					}
					for j := 0; j < int(node.ChildCount()); j++ {
						token := node.Child(j)
						if !token.IsNamed() && !token.IsMissing() && token.StartByte() == repair.operatorStart && token.EndByte() == repair.operatorEnd && token.Content(source) == string(source[repair.operatorStart:repair.operatorEnd]) {
							found[i] = true
						}
					}
				}
			}
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			visit(node.NamedChild(i))
		}
	}
	visit(root)
	for _, matched := range found {
		if !matched {
			return false
		}
	}
	return len(found) > 0
}
