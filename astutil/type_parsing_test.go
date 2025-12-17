package astutil

import (
	"context"
	"go/ast"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

// parseJavaType parses a Java source file and finds the type node for a field declaration.
func parseJavaType(t *testing.T, source string) *sitter.Node {
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(source))
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}
	return tree.RootNode()
}

// findNode recursively searches for a node of a given type
func findNode(node *sitter.Node, typeName string) *sitter.Node {
	if node.Type() == typeName {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		found := findNode(node.Child(i), typeName)
		if found != nil {
			return found
		}
	}
	return nil
}

func TestParseTypeWithTypeParams_TypeIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		typeParams []string
		wantType   string // "ident" for *ast.Ident, "star" for *ast.StarExpr
		wantName   string
	}{
		{
			name:       "type_identifier not in typeParams gets wrapped in pointer",
			source:     "class C { SomeClass field; }",
			typeParams: nil,
			wantType:   "star",
			wantName:   "SomeClass",
		},
		{
			name:       "type_identifier T without typeParams gets wrapped in pointer",
			source:     "class C { T field; }",
			typeParams: nil,
			wantType:   "star",
			wantName:   "T",
		},
		{
			name:       "type_identifier T with matching typeParam stays as bare identifier",
			source:     "class C { T field; }",
			typeParams: []string{"T"},
			wantType:   "ident",
			wantName:   "T",
		},
		{
			name:       "type_identifier K with multiple typeParams matches correctly",
			source:     "class C { K field; }",
			typeParams: []string{"K", "V"},
			wantType:   "ident",
			wantName:   "K",
		},
		{
			name:       "type_identifier V with multiple typeParams matches correctly",
			source:     "class C { V field; }",
			typeParams: []string{"K", "V"},
			wantType:   "ident",
			wantName:   "V",
		},
		{
			name:       "type_identifier X not in typeParams list gets wrapped",
			source:     "class C { X field; }",
			typeParams: []string{"T", "U"},
			wantType:   "star",
			wantName:   "X",
		},
		{
			name:       "String type becomes string (primitive)",
			source:     "class C { String field; }",
			typeParams: nil,
			wantType:   "ident",
			wantName:   "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := parseJavaType(t, tt.source)
			typeNode := findNode(root, "type_identifier")
			if typeNode == nil {
				t.Fatal("Could not find type_identifier node")
			}

			result := ParseTypeWithTypeParams(typeNode, []byte(tt.source), tt.typeParams)

			switch tt.wantType {
			case "ident":
				ident, ok := result.(*ast.Ident)
				if !ok {
					t.Fatalf("Expected *ast.Ident, got %T", result)
				}
				if ident.Name != tt.wantName {
					t.Errorf("Expected name '%s', got '%s'", tt.wantName, ident.Name)
				}
			case "star":
				star, ok := result.(*ast.StarExpr)
				if !ok {
					t.Fatalf("Expected *ast.StarExpr, got %T", result)
				}
				ident, ok := star.X.(*ast.Ident)
				if !ok {
					t.Fatalf("Expected StarExpr.X to be *ast.Ident, got %T", star.X)
				}
				if ident.Name != tt.wantName {
					t.Errorf("Expected name '%s', got '%s'", tt.wantName, ident.Name)
				}
			}
		})
	}
}

func TestParseTypeWithTypeParams_GenericType(t *testing.T) {
	tests := []struct {
		name          string
		source        string
		typeParams    []string
		wantBaseName  string
		wantArgCount  int
		wantFirstArg  string
		wantArgIsStar bool // true if type argument should be wrapped in star
	}{
		{
			name:          "generic type List<String>",
			source:        "class C { List<String> field; }",
			typeParams:    nil,
			wantBaseName:  "List",
			wantArgCount:  1,
			wantFirstArg:  "string",
			wantArgIsStar: false,
		},
		{
			name:          "generic type List<T> with T in typeParams",
			source:        "class C { List<T> field; }",
			typeParams:    []string{"T"},
			wantBaseName:  "List",
			wantArgCount:  1,
			wantFirstArg:  "T",
			wantArgIsStar: false,
		},
		{
			name:          "generic type Map<K, V> with K,V in typeParams",
			source:        "class C { Map<K, V> field; }",
			typeParams:    []string{"K", "V"},
			wantBaseName:  "Map",
			wantArgCount:  2,
			wantFirstArg:  "K",
			wantArgIsStar: false,
		},
		{
			name:          "generic type Box<SomeClass> wraps type arg in star",
			source:        "class C { Box<SomeClass> field; }",
			typeParams:    nil,
			wantBaseName:  "Box",
			wantArgCount:  1,
			wantFirstArg:  "SomeClass",
			wantArgIsStar: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := parseJavaType(t, tt.source)
			typeNode := findNode(root, "generic_type")
			if typeNode == nil {
				t.Fatal("Could not find generic_type node")
			}

			result := ParseTypeWithTypeParams(typeNode, []byte(tt.source), tt.typeParams)

			// Result should be *ast.StarExpr wrapping an IndexExpr or IndexListExpr
			star, ok := result.(*ast.StarExpr)
			if !ok {
				t.Fatalf("Expected *ast.StarExpr, got %T", result)
			}

			var baseName string
			var typeArgs []ast.Expr

			switch indexExpr := star.X.(type) {
			case *ast.IndexExpr:
				ident, ok := indexExpr.X.(*ast.Ident)
				if !ok {
					t.Fatalf("Expected IndexExpr.X to be *ast.Ident, got %T", indexExpr.X)
				}
				baseName = ident.Name
				typeArgs = []ast.Expr{indexExpr.Index}
			case *ast.IndexListExpr:
				ident, ok := indexExpr.X.(*ast.Ident)
				if !ok {
					t.Fatalf("Expected IndexListExpr.X to be *ast.Ident, got %T", indexExpr.X)
				}
				baseName = ident.Name
				typeArgs = indexExpr.Indices
			default:
				t.Fatalf("Expected star.X to be *ast.IndexExpr or *ast.IndexListExpr, got %T", star.X)
			}

			if baseName != tt.wantBaseName {
				t.Errorf("Expected base name '%s', got '%s'", tt.wantBaseName, baseName)
			}

			if len(typeArgs) != tt.wantArgCount {
				t.Errorf("Expected %d type arguments, got %d", tt.wantArgCount, len(typeArgs))
			}

			if len(typeArgs) > 0 {
				firstArg := typeArgs[0]
				if tt.wantArgIsStar {
					star, ok := firstArg.(*ast.StarExpr)
					if !ok {
						t.Fatalf("Expected first type arg to be *ast.StarExpr, got %T", firstArg)
					}
					ident, ok := star.X.(*ast.Ident)
					if !ok {
						t.Fatalf("Expected StarExpr.X to be *ast.Ident, got %T", star.X)
					}
					if ident.Name != tt.wantFirstArg {
						t.Errorf("Expected first arg name '%s', got '%s'", tt.wantFirstArg, ident.Name)
					}
				} else {
					ident, ok := firstArg.(*ast.Ident)
					if !ok {
						t.Fatalf("Expected first type arg to be *ast.Ident, got %T", firstArg)
					}
					if ident.Name != tt.wantFirstArg {
						t.Errorf("Expected first arg name '%s', got '%s'", tt.wantFirstArg, ident.Name)
					}
				}
			}
		})
	}
}

func TestParseTypeWithTypeParams_ArrayType(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		typeParams []string
		wantEltStr string // expected element type as string representation
	}{
		{
			name:       "array of primitive int",
			source:     "class C { int[] field; }",
			typeParams: nil,
			wantEltStr: "int32",
		},
		{
			name:       "array of type parameter T",
			source:     "class C { T[] field; }",
			typeParams: []string{"T"},
			wantEltStr: "T",
		},
		{
			name:       "array of generic type List<T>",
			source:     "class C<T> { List<T>[] field; }",
			typeParams: []string{"T"},
			wantEltStr: "*List[T]",
		},
		{
			name:       "nested array T[][]",
			source:     "class C { T[][] field; }",
			typeParams: []string{"T"},
			wantEltStr: "[]T", // outer array contains inner []T
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := parseJavaType(t, tt.source)
			typeNode := findNode(root, "array_type")
			if typeNode == nil {
				t.Fatal("Could not find array_type node")
			}

			result := ParseTypeWithTypeParams(typeNode, []byte(tt.source), tt.typeParams)

			arrType, ok := result.(*ast.ArrayType)
			if !ok {
				t.Fatalf("Expected *ast.ArrayType, got %T", result)
			}

			// Check element type based on expected string
			switch tt.wantEltStr {
			case "int32":
				ident, ok := arrType.Elt.(*ast.Ident)
				if !ok {
					t.Fatalf("Expected element to be *ast.Ident, got %T", arrType.Elt)
				}
				if ident.Name != "int32" {
					t.Errorf("Expected element name 'int32', got '%s'", ident.Name)
				}
			case "T":
				ident, ok := arrType.Elt.(*ast.Ident)
				if !ok {
					t.Fatalf("Expected element to be *ast.Ident, got %T", arrType.Elt)
				}
				if ident.Name != "T" {
					t.Errorf("Expected element name 'T', got '%s'", ident.Name)
				}
			case "[]T":
				innerArr, ok := arrType.Elt.(*ast.ArrayType)
				if !ok {
					t.Fatalf("Expected element to be *ast.ArrayType, got %T", arrType.Elt)
				}
				ident, ok := innerArr.Elt.(*ast.Ident)
				if !ok {
					t.Fatalf("Expected inner array element to be *ast.Ident, got %T", innerArr.Elt)
				}
				if ident.Name != "T" {
					t.Errorf("Expected inner element name 'T', got '%s'", ident.Name)
				}
			case "*List[T]":
				star, ok := arrType.Elt.(*ast.StarExpr)
				if !ok {
					t.Fatalf("Expected element to be *ast.StarExpr, got %T", arrType.Elt)
				}
				indexExpr, ok := star.X.(*ast.IndexExpr)
				if !ok {
					t.Fatalf("Expected star.X to be *ast.IndexExpr, got %T", star.X)
				}
				baseIdent, ok := indexExpr.X.(*ast.Ident)
				if !ok {
					t.Fatalf("Expected IndexExpr.X to be *ast.Ident, got %T", indexExpr.X)
				}
				if baseIdent.Name != "List" {
					t.Errorf("Expected base ident 'List', got '%s'", baseIdent.Name)
				}
				argIdent, ok := indexExpr.Index.(*ast.Ident)
				if !ok {
					t.Fatalf("Expected type argument to be *ast.Ident, got %T", indexExpr.Index)
				}
				if argIdent.Name != "T" {
					t.Errorf("Expected type arg 'T', got '%s'", argIdent.Name)
				}
			}
		})
	}
}

func TestParseTypeWithTypeParams_PrimitiveTypes(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantName string
	}{
		{"int becomes int32", "class C { int field; }", "int32"},
		{"short becomes int16", "class C { short field; }", "int16"},
		{"long becomes int64", "class C { long field; }", "int64"},
		{"char becomes rune", "class C { char field; }", "rune"},
		{"byte stays byte", "class C { byte field; }", "byte"},
		{"float becomes float32", "class C { float field; }", "float32"},
		{"double becomes float64", "class C { double field; }", "float64"},
		{"boolean becomes bool", "class C { boolean field; }", "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := parseJavaType(t, tt.source)

			// Find the field declaration type node
			fieldDecl := findNode(root, "field_declaration")
			if fieldDecl == nil {
				t.Fatal("Could not find field_declaration")
			}

			// Get the type child (first named child after modifiers if any)
			var typeNode *sitter.Node
			for i := 0; i < int(fieldDecl.NamedChildCount()); i++ {
				child := fieldDecl.NamedChild(i)
				switch child.Type() {
				case "integral_type", "floating_point_type", "boolean_type":
					typeNode = child
				}
				if typeNode != nil {
					break
				}
			}

			if typeNode == nil {
				t.Fatal("Could not find type node")
			}

			result := ParseTypeWithTypeParams(typeNode, []byte(tt.source), nil)

			ident, ok := result.(*ast.Ident)
			if !ok {
				t.Fatalf("Expected *ast.Ident, got %T", result)
			}
			if ident.Name != tt.wantName {
				t.Errorf("Expected '%s', got '%s'", tt.wantName, ident.Name)
			}
		})
	}
}

func TestExtractTypeArguments(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantArgs []string
	}{
		{
			name:     "single type argument",
			source:   "class C { List<String> field; }",
			wantArgs: []string{"String"},
		},
		{
			name:     "multiple type arguments",
			source:   "class C { Map<String, Integer> field; }",
			wantArgs: []string{"String", "Integer"},
		},
		{
			name:     "type parameter as argument",
			source:   "class C { List<T> field; }",
			wantArgs: []string{"T"},
		},
		{
			name:     "diamond operator (empty type args)",
			source:   "class C { void m() { new List<>(); } }",
			wantArgs: []string{}, // Diamond operator has no type arguments
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := parseJavaType(t, tt.source)
			typeNode := findNode(root, "generic_type")
			if typeNode == nil {
				t.Fatal("Could not find generic_type node")
			}

			result := ExtractTypeArguments(typeNode, []byte(tt.source))

			if len(result) != len(tt.wantArgs) {
				t.Errorf("Expected %d type arguments, got %d: %v", len(tt.wantArgs), len(result), result)
				return
			}

			for i, want := range tt.wantArgs {
				if result[i] != want {
					t.Errorf("Type arg %d: expected '%s', got '%s'", i, want, result[i])
				}
			}
		})
	}
}

func TestExtractTypeArguments_NonGenericType(t *testing.T) {
	source := "class C { String field; }"
	root := parseJavaType(t, source)
	typeNode := findNode(root, "type_identifier")
	if typeNode == nil {
		t.Fatal("Could not find type_identifier node")
	}

	result := ExtractTypeArguments(typeNode, []byte(source))

	if result != nil {
		t.Errorf("Expected nil for non-generic type, got %v", result)
	}
}
