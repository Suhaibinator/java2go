package astutil

import (
	"fmt"
	"go/ast"

	sitter "github.com/smacker/go-tree-sitter"
)

// ParseType parses a Java type node and converts it to a Go AST expression.
// This version does not handle type parameters - use ParseTypeWithTypeParams for generic contexts.
func ParseType(node *sitter.Node, source []byte) ast.Expr {
	return ParseTypeWithTypeParams(node, source, nil)
}

// ParseTypeWithTypeParams parses a Java type node and converts it to a Go AST expression.
// typeParams is a list of type parameter names that should not be wrapped in pointers.
func ParseTypeWithTypeParams(node *sitter.Node, source []byte, typeParams []string) ast.Expr {
	// Helper function to check if a name is a type parameter
	isTypeParam := func(name string) bool {
		for _, tp := range typeParams {
			if tp == name {
				return true
			}
		}
		return false
	}

	switch node.Type() {
	case "integral_type":
		switch node.Child(0).Type() {
		case "int":
			return &ast.Ident{Name: "int32"}
		case "short":
			return &ast.Ident{Name: "int16"}
		case "long":
			return &ast.Ident{Name: "int64"}
		case "char":
			return &ast.Ident{Name: "rune"}
		case "byte":
			return &ast.Ident{Name: node.Content(source)}
		}

		panic(fmt.Errorf("Unknown integral type: %v", node.Child(0).Type()))
	case "floating_point_type": // Can be either `float` or `double`
		switch node.Child(0).Type() {
		case "float":
			return &ast.Ident{Name: "float32"}
		case "double":
			return &ast.Ident{Name: "float64"}
		}

		panic(fmt.Errorf("Unknown float type: %v", node.Child(0).Type()))
	case "void_type":
		return &ast.Ident{}
	case "boolean_type":
		return &ast.Ident{Name: "bool"}
	case "generic_type":
		// A generic type is any type that is of the form GenericType<T>
		// Extract the base type and type arguments
		baseName := node.NamedChild(0).Content(source)

		// Find the type_arguments node
		var typeArgs []ast.Expr
		for i := 0; i < int(node.NamedChildCount()); i++ {
			child := node.NamedChild(i)
			if child.Type() == "type_arguments" {
				// Parse each type argument
				for j := 0; j < int(child.NamedChildCount()); j++ {
					argNode := child.NamedChild(j)
					typeArgs = append(typeArgs, ParseTypeWithTypeParams(argNode, source, typeParams))
				}
				break
			}
		}

		// If we have type arguments, create an IndexExpr or IndexListExpr
		// The pointer wraps the entire indexed expression: *List[T], not (*List)[T]
		if len(typeArgs) > 0 {
			baseIdent := &ast.Ident{Name: baseName}
			var indexedExpr ast.Expr
			if len(typeArgs) == 1 {
				indexedExpr = &ast.IndexExpr{
					X:     baseIdent,
					Index: typeArgs[0],
				}
			} else {
				// Multiple type arguments use IndexListExpr
				indexedExpr = &ast.IndexListExpr{
					X:       baseIdent,
					Indices: typeArgs,
				}
			}
			return &ast.StarExpr{X: indexedExpr}
		}

		// No type arguments, just return the base type as a pointer
		return &ast.StarExpr{X: &ast.Ident{Name: baseName}}
	case "array_type":
		return &ast.ArrayType{Elt: ParseTypeWithTypeParams(node.NamedChild(0), source, typeParams)}
	case "type_identifier": // Any reference type
		typeName := node.Content(source)

		// Special case for strings, because in Go, these are primitive types
		if typeName == "String" {
			return &ast.Ident{Name: "string"}
		}

		// If this is a type parameter, don't wrap it in a pointer
		if isTypeParam(typeName) {
			return &ast.Ident{Name: typeName}
		}

		return &ast.StarExpr{
			X: &ast.Ident{Name: typeName},
		}
	case "scoped_type_identifier":
		// This contains a reference to the type of a nested class
		// Ex: LinkedList.Node
		return &ast.StarExpr{X: &ast.Ident{Name: node.Content(source)}}
	}
	panic("Unknown type to convert: " + node.Type())
}

// ExtractTypeArguments extracts type argument strings from a generic_type node.
// Returns empty slice if node is not a generic type or has no type arguments.
func ExtractTypeArguments(node *sitter.Node, source []byte) []string {
	if node.Type() != "generic_type" {
		return nil
	}

	var typeArgs []string
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "type_arguments" {
			for j := 0; j < int(child.NamedChildCount()); j++ {
				argNode := child.NamedChild(j)
				typeArgs = append(typeArgs, argNode.Content(source))
			}
			break
		}
	}
	return typeArgs
}
