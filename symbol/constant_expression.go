package symbol

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func javaConstantVariableType(javaType string) bool {
	javaType = strings.TrimSpace(javaType)
	if strings.HasSuffix(javaType, "[]") {
		return false
	}
	if index := strings.LastIndex(javaType, "."); index >= 0 {
		javaType = javaType[index+1:]
	}
	switch javaType {
	case "byte", "short", "int", "long", "char", "float", "double", "boolean", "String":
		return true
	default:
		return false
	}
}

// javaConstantExpression is deliberately syntax-directed. Java has already
// rejected illegal operand/type combinations before this project is normally
// invoked; this predicate identifies only constructs admitted by JLS 15.29 and
// known constant variables already discovered in the same class.
func javaConstantExpression(node *sitter.Node, source []byte, scope *ClassScope) bool {
	if node == nil {
		return false
	}
	switch node.Type() {
	case "decimal_integer_literal", "hex_integer_literal", "octal_integer_literal", "binary_integer_literal",
		"decimal_floating_point_literal", "hex_floating_point_literal",
		"character_literal", "string_literal", "true", "false":
		return true
	case "parenthesized_expression", "unary_expression":
		return node.NamedChildCount() == 1 && javaConstantExpression(node.NamedChild(0), source, scope)
	case "cast_expression":
		if node.NamedChildCount() != 2 || !javaConstantVariableType(node.NamedChild(0).Content(source)) {
			return false
		}
		return javaConstantExpression(node.NamedChild(1), source, scope)
	case "binary_expression":
		if node.ChildCount() < 3 || node.Child(1).Content(source) == "instanceof" {
			return false
		}
		return javaConstantExpression(node.Child(0), source, scope) &&
			javaConstantExpression(node.Child(2), source, scope)
	case "ternary_expression":
		return node.NamedChildCount() == 3 &&
			javaConstantExpression(node.NamedChild(0), source, scope) &&
			javaConstantExpression(node.NamedChild(1), source, scope) &&
			javaConstantExpression(node.NamedChild(2), source, scope)
	case "identifier":
		if scope == nil {
			return false
		}
		field := scope.FindFieldByName(node.Content(source))
		return field != nil && field.IsCompileTimeConstant
	case "field_access":
		if scope == nil {
			return false
		}
		object := node.ChildByFieldName("object")
		fieldNode := node.ChildByFieldName("field")
		if object == nil || fieldNode == nil || object.Content(source) != scope.Class.OriginalName {
			return false
		}
		field := scope.FindFieldByName(fieldNode.Content(source))
		return field != nil && field.IsCompileTimeConstant
	default:
		return false
	}
}
