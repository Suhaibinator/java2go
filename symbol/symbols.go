package symbol

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// Go reserved keywords that are not Java keywords, and create invalid code
var reservedKeywords = []string{"chan", "defer", "fallthrough", "func", "go", "map", "range", "select", "struct", "type"}

// IsReserved tests if a given identifier conflicts with a Go reserved keyword
func IsReserved(name string) bool {
	for _, keyword := range reservedKeywords {
		if keyword == name {
			return true
		}
	}
	return false
}

// TypeOfLiteral returns the corresponding type for a Java literal
func TypeOfLiteral(node *sitter.Node, source []byte) string {
	var originalType string

	switch node.Type() {
	case "decimal_integer_literal":
		switch node.Content(source)[len(node.Content(source))-1] {
		case 'L':
			originalType = "long"
		default:
			originalType = "int"
		}
	case "hex_integer_literal":
		panic("here")
	case "decimal_floating_point_literal":
		switch node.Content(source)[len(node.Content(source))-1] {
		case 'D':
			originalType = "double"
		default:
			originalType = "float"
		}
	case "string_literal":
		originalType = "String"
	case "character_literal":
		originalType = "char"
	}

	return originalType
}

// ResolveDefinition resolves a given definition, given its scope in the file
// It returns `true` on a successful resolution, or `false` otherwise
//
// Resolving a definition means that the type of the file is matched up with the type defined
// in the local scope or otherwise
func ResolveDefinition(definition *Definition, fileScope *FileScope) bool {
	selected := registeredSourceDefinitionClass(definition, fileScope)
	if selected == nil || selected.Class == nil || selected.Class.Name == "" {
		return false
	}
	definition.Type = "*" + selected.Class.Name
	return true
}

// ResolveChildren recursively resolves a definition and all of its children
// It returns true if all definitions were resolved correctly, and false otherwise
func ResolveChildren(definition *Definition, fileScope *FileScope) bool {
	result := ResolveDefinition(definition, fileScope)
	for _, child := range definition.Children {
		result = ResolveChildren(child, fileScope) && result
	}
	return result
}
