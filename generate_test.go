package main

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestGenStructWithTypeParams_NoTypeParams(t *testing.T) {
	fields := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "value"}},
				Type:  &ast.Ident{Name: "int"},
			},
		},
	}

	result := GenStructWithTypeParams("MyStruct", fields, nil)

	genDecl, ok := result.(*ast.GenDecl)
	if !ok {
		t.Fatalf("Expected *ast.GenDecl, got %T", result)
	}

	if genDecl.Tok != token.TYPE {
		t.Errorf("Expected token.TYPE, got %v", genDecl.Tok)
	}

	if len(genDecl.Specs) != 1 {
		t.Fatalf("Expected 1 spec, got %d", len(genDecl.Specs))
	}

	typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
	if !ok {
		t.Fatalf("Expected *ast.TypeSpec, got %T", genDecl.Specs[0])
	}

	if typeSpec.Name.Name != "MyStruct" {
		t.Errorf("Expected name 'MyStruct', got '%s'", typeSpec.Name.Name)
	}

	// No type params should be present
	if typeSpec.TypeParams != nil {
		t.Error("Expected TypeParams to be nil for non-generic struct")
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		t.Fatalf("Expected *ast.StructType, got %T", typeSpec.Type)
	}

	if len(structType.Fields.List) != 1 {
		t.Errorf("Expected 1 field, got %d", len(structType.Fields.List))
	}
}

func TestGenStructWithTypeParams_SingleTypeParam(t *testing.T) {
	fields := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "Value"}},
				Type:  &ast.Ident{Name: "T"},
			},
		},
	}

	typeParams := []symbol.TypeParam{{Name: "T"}}

	result := GenStructWithTypeParams("Box", fields, typeParams)

	genDecl, ok := result.(*ast.GenDecl)
	if !ok {
		t.Fatalf("Expected *ast.GenDecl, got %T", result)
	}

	typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
	if !ok {
		t.Fatalf("Expected *ast.TypeSpec, got %T", genDecl.Specs[0])
	}

	if typeSpec.Name.Name != "Box" {
		t.Errorf("Expected name 'Box', got '%s'", typeSpec.Name.Name)
	}

	// Check type params
	if typeSpec.TypeParams == nil {
		t.Fatal("Expected TypeParams to be non-nil")
	}

	if len(typeSpec.TypeParams.List) != 1 {
		t.Fatalf("Expected 1 type param, got %d", len(typeSpec.TypeParams.List))
	}

	typeParam := typeSpec.TypeParams.List[0]
	if len(typeParam.Names) != 1 || typeParam.Names[0].Name != "T" {
		t.Errorf("Expected type param name 'T', got %v", typeParam.Names)
	}

	// Constraint should be "any"
	constraint, ok := typeParam.Type.(*ast.Ident)
	if !ok {
		t.Fatalf("Expected constraint to be *ast.Ident, got %T", typeParam.Type)
	}
	if constraint.Name != "any" {
		t.Errorf("Expected constraint 'any', got '%s'", constraint.Name)
	}
}

func TestGenStructWithTypeParams_MultipleTypeParams(t *testing.T) {
	fields := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "Key"}},
				Type:  &ast.Ident{Name: "K"},
			},
			{
				Names: []*ast.Ident{{Name: "Value"}},
				Type:  &ast.Ident{Name: "V"},
			},
		},
	}

	typeParams := []symbol.TypeParam{{Name: "K"}, {Name: "V"}}

	result := GenStructWithTypeParams("Pair", fields, typeParams)

	genDecl, ok := result.(*ast.GenDecl)
	if !ok {
		t.Fatalf("Expected *ast.GenDecl, got %T", result)
	}

	typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
	if !ok {
		t.Fatalf("Expected *ast.TypeSpec, got %T", genDecl.Specs[0])
	}

	// Check type params
	if typeSpec.TypeParams == nil {
		t.Fatal("Expected TypeParams to be non-nil")
	}

	if len(typeSpec.TypeParams.List) != 2 {
		t.Fatalf("Expected 2 type params, got %d", len(typeSpec.TypeParams.List))
	}

	// Verify K and V params with "any" constraints
	expectedParams := []string{"K", "V"}
	for i, param := range typeSpec.TypeParams.List {
		if len(param.Names) != 1 || param.Names[0].Name != expectedParams[i] {
			t.Errorf("Type param %d: expected name '%s', got %v", i, expectedParams[i], param.Names)
		}
		constraint, ok := param.Type.(*ast.Ident)
		if !ok || constraint.Name != "any" {
			t.Errorf("Type param %d: expected constraint 'any', got %v", i, param.Type)
		}
	}

	// Verify printed output
	var buf bytes.Buffer
	fset := token.NewFileSet()
	if err := printer.Fprint(&buf, fset, result); err != nil {
		t.Fatalf("Failed to print AST: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "Pair[K any, V any]") {
		t.Errorf("Expected 'Pair[K any, V any]' in output, got:\n%s", output)
	}
}

func TestGenStructWithTypeParams_TypeParamBounds(t *testing.T) {
	fields := &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "value"}}, Type: &ast.Ident{Name: "T"}}}}
	typeParams := []symbol.TypeParam{{
		Name:   "T",
		Bounds: []symbol.JavaType{{Original: "Number"}, {Original: "Comparable<T>"}},
	}}

	result := GenStructWithTypeParams("Bounded", fields, typeParams)

	genDecl, ok := result.(*ast.GenDecl)
	if !ok {
		t.Fatalf("Expected *ast.GenDecl, got %T", result)
	}

	typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
	if !ok {
		t.Fatalf("Expected *ast.TypeSpec, got %T", genDecl.Specs[0])
	}

	if len(typeSpec.TypeParams.List) != 1 {
		t.Fatalf("Expected 1 type param, got %d", len(typeSpec.TypeParams.List))
	}

	constraint, ok := typeSpec.TypeParams.List[0].Type.(*ast.InterfaceType)
	if !ok {
		t.Fatalf("Expected constraint to be *ast.InterfaceType, got %T", typeSpec.TypeParams.List[0].Type)
	}

	if got := len(constraint.Methods.List); got != 2 {
		t.Fatalf("Expected 2 embedded bounds, got %d", got)
	}

	firstBound, ok := constraint.Methods.List[0].Type.(*ast.StarExpr)
	if !ok {
		t.Fatalf("Expected first bound to be *ast.StarExpr, got %T", constraint.Methods.List[0].Type)
	}
	if ident, ok := firstBound.X.(*ast.Ident); !ok || ident.Name != "Number" {
		t.Fatalf("Expected first bound identifier 'Number', got %v", firstBound.X)
	}

	secondBound, ok := constraint.Methods.List[1].Type.(*ast.StarExpr)
	if !ok {
		t.Fatalf("Expected second bound to be *ast.StarExpr, got %T", constraint.Methods.List[1].Type)
	}
	indexExpr, ok := secondBound.X.(*ast.IndexExpr)
	if !ok {
		t.Fatalf("Expected second bound to be *ast.IndexExpr inside *ast.StarExpr, got %T", secondBound.X)
	}
	if ident, ok := indexExpr.X.(*ast.Ident); !ok || ident.Name != "Comparable" {
		t.Fatalf("Expected base identifier 'Comparable', got %v", indexExpr.X)
	}
	if arg, ok := indexExpr.Index.(*ast.Ident); !ok || arg.Name != "T" {
		t.Fatalf("Expected type argument 'T', got %v", indexExpr.Index)
	}
}

func TestGenFuncDeclWithTypeParams_NoTypeParams(t *testing.T) {
	params := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "x"}},
				Type:  &ast.Ident{Name: "int"},
			},
		},
	}
	results := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.Ident{Name: "int"}},
		},
	}
	body := &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{&ast.Ident{Name: "x"}},
			},
		},
	}

	result := GenFuncDeclWithTypeParams("Identity", nil, params, results, body)

	if result.Name.Name != "Identity" {
		t.Errorf("Expected name 'Identity', got '%s'", result.Name.Name)
	}

	// No type params
	if result.Type.TypeParams != nil {
		t.Error("Expected TypeParams to be nil for non-generic function")
	}

	// Check params and results
	if len(result.Type.Params.List) != 1 {
		t.Errorf("Expected 1 param, got %d", len(result.Type.Params.List))
	}
	if len(result.Type.Results.List) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result.Type.Results.List))
	}
}

func TestGenFuncDeclWithTypeParams_SingleTypeParam(t *testing.T) {
	params := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "value"}},
				Type:  &ast.Ident{Name: "T"},
			},
		},
	}
	results := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: &ast.IndexExpr{
				X:     &ast.Ident{Name: "Box"},
				Index: &ast.Ident{Name: "T"},
			}}},
		},
	}
	body := &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{&ast.Ident{Name: "nil"}},
			},
		},
	}

	typeParams := []symbol.TypeParam{{Name: "T"}}

	result := GenFuncDeclWithTypeParams("ConstructBox", typeParams, params, results, body)

	if result.Name.Name != "ConstructBox" {
		t.Errorf("Expected name 'ConstructBox', got '%s'", result.Name.Name)
	}

	// Check type params
	if result.Type.TypeParams == nil {
		t.Fatal("Expected TypeParams to be non-nil")
	}

	if len(result.Type.TypeParams.List) != 1 {
		t.Fatalf("Expected 1 type param, got %d", len(result.Type.TypeParams.List))
	}

	typeParam := result.Type.TypeParams.List[0]
	if len(typeParam.Names) != 1 || typeParam.Names[0].Name != "T" {
		t.Errorf("Expected type param name 'T', got %v", typeParam.Names)
	}

	constraint, ok := typeParam.Type.(*ast.Ident)
	if !ok || constraint.Name != "any" {
		t.Errorf("Expected constraint 'any', got %v", typeParam.Type)
	}
}

func TestGenFuncDeclWithTypeParams_MultipleTypeParams(t *testing.T) {
	params := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "key"}},
				Type:  &ast.Ident{Name: "K"},
			},
			{
				Names: []*ast.Ident{{Name: "value"}},
				Type:  &ast.Ident{Name: "V"},
			},
		},
	}
	results := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: &ast.IndexListExpr{
				X:       &ast.Ident{Name: "Pair"},
				Indices: []ast.Expr{&ast.Ident{Name: "K"}, &ast.Ident{Name: "V"}},
			}}},
		},
	}
	body := &ast.BlockStmt{}

	typeParams := []symbol.TypeParam{{Name: "K"}, {Name: "V"}}

	result := GenFuncDeclWithTypeParams("ConstructPair", typeParams, params, results, body)

	// Check type params
	if result.Type.TypeParams == nil {
		t.Fatal("Expected TypeParams to be non-nil")
	}

	if len(result.Type.TypeParams.List) != 2 {
		t.Fatalf("Expected 2 type params, got %d", len(result.Type.TypeParams.List))
	}

	// Verify K and V params with "any" constraints
	expectedParams := []string{"K", "V"}
	for i, param := range result.Type.TypeParams.List {
		if len(param.Names) != 1 || param.Names[0].Name != expectedParams[i] {
			t.Errorf("Type param %d: expected name '%s', got %v", i, expectedParams[i], param.Names)
		}
		constraint, ok := param.Type.(*ast.Ident)
		if !ok || constraint.Name != "any" {
			t.Errorf("Type param %d: expected constraint 'any', got %v", i, param.Type)
		}
	}

	// Verify printed output
	var buf bytes.Buffer
	fset := token.NewFileSet()
	if err := printer.Fprint(&buf, fset, result); err != nil {
		t.Fatalf("Failed to print AST: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "ConstructPair[K any, V any]") {
		t.Errorf("Expected 'ConstructPair[K any, V any]' in output, got:\n%s", output)
	}
}

func TestGenStruct_DelegatesCorrectly(t *testing.T) {
	fields := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "Field"}},
				Type:  &ast.Ident{Name: "int"},
			},
		},
	}

	result := GenStruct("Simple", fields)

	genDecl, ok := result.(*ast.GenDecl)
	if !ok {
		t.Fatalf("Expected *ast.GenDecl, got %T", result)
	}

	typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
	if !ok {
		t.Fatalf("Expected *ast.TypeSpec, got %T", genDecl.Specs[0])
	}

	// GenStruct should produce a struct without type params (delegates to GenStructWithTypeParams with nil)
	if typeSpec.TypeParams != nil {
		t.Error("GenStruct should not add type params")
	}

	if typeSpec.Name.Name != "Simple" {
		t.Errorf("Expected name 'Simple', got '%s'", typeSpec.Name.Name)
	}
}

func TestShortName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Test", "tt"},
		{"Box", "bx"},
		{"LinkedList", "lt"},
		{"A", "aa"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ShortName(tt.input)
			if got != tt.want {
				t.Errorf("ShortName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStrToToken(t *testing.T) {
	tests := []struct {
		input string
		want  token.Token
	}{
		{"+", token.ADD},
		{"-", token.SUB},
		{"*", token.MUL},
		{"==", token.EQL},
		{"!=", token.NEQ},
		{"&&", token.LAND},
		{"||", token.LOR},
		{"++", token.INC},
		{"--", token.DEC},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := StrToToken(tt.input)
			if got != tt.want {
				t.Errorf("StrToToken(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestStrToToken_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for unknown token")
		}
	}()
	StrToToken("unknown_token")
}
