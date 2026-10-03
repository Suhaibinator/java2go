package transpiler

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestGeneratedAllocationIdentityStorage(t *testing.T) {
	empty := &ast.StructType{Fields: &ast.FieldList{}}
	cases := []struct {
		name   string
		fields []ast.Expr
		marker bool
	}{
		{"empty", nil, true},
		{"zero_array", []ast.Expr{&ast.ArrayType{Len: &ast.BasicLit{Kind: token.INT, Value: "0"}, Elt: ast.NewIdent("byte")}}, true},
		{"positive_array_of_zero", []ast.Expr{&ast.ArrayType{Len: &ast.BasicLit{Kind: token.INT, Value: "1"}, Elt: empty}}, true},
		{"zero_embedded_value", []ast.Expr{empty}, true},
		{"unknown_named", []ast.Expr{ast.NewIdent("Opaque")}, true},
		{"unknown_parameter", []ast.Expr{ast.NewIdent("T")}, true},
		{"pointer", []ast.Expr{&ast.StarExpr{X: ast.NewIdent("Empty")}}, false},
		{"positive_pointer_array", []ast.Expr{&ast.ArrayType{Len: &ast.BasicLit{Kind: token.INT, Value: "1"}, Elt: &ast.StarExpr{X: empty}}}, false},
		{"positive_array_of_nonzero_struct", []ast.Expr{&ast.ArrayType{Len: &ast.BasicLit{Kind: token.INT, Value: "1"}, Elt: &ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: empty}}}}}}}, false},
		{"interface", []ast.Expr{&ast.InterfaceType{Methods: &ast.FieldList{}}}, false},
		{"function", []ast.Expr{&ast.FuncType{Params: &ast.FieldList{}}}, false},
		{"slice", []ast.Expr{&ast.ArrayType{Elt: empty}}, false},
		{"builtin_scalar_array", []ast.Expr{&ast.ArrayType{Len: &ast.BasicLit{Kind: token.INT, Value: "1"}, Elt: ast.NewIdent("byte")}}, false},
		{"unknown_plus_builtin_scalar", []ast.Expr{ast.NewIdent("Opaque"), ast.NewIdent("byte")}, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fields := &ast.FieldList{}
			for _, field := range test.fields {
				fields.List = append(fields.List, &ast.Field{Type: field})
			}
			generated := genStructWithTypeParamsInContext("Probe", fields, nil, Ctx{}).(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.StructType)
			want := len(test.fields)
			if test.marker {
				want++
			}
			if len(generated.Fields.List) != want {
				t.Fatalf("generated fields=%d, want %d (marker=%v)", len(generated.Fields.List), want, test.marker)
			}
			if len(fields.List) != len(test.fields) {
				t.Fatal("generation mutated caller's field list")
			}
			if test.marker {
				marker := generated.Fields.List[len(generated.Fields.List)-1]
				if len(marker.Names) != 1 || marker.Names[0].Name != "_" {
					t.Fatalf("marker must be blank, got %#v", marker.Names)
				}
				pointer, ok := marker.Type.(*ast.StarExpr)
				if !ok {
					t.Fatalf("marker type=%#v, want pointer", marker.Type)
				}
				empty, ok := pointer.X.(*ast.StructType)
				if !ok || empty.Fields == nil || len(empty.Fields.List) != 0 {
					t.Fatalf("marker pointee=%#v, want unnamed empty struct", pointer.X)
				}
			}
		})
	}
}

func TestGeneratedAllocationIdentityBuiltinShadows(t *testing.T) {
	cases := []struct {
		name, structName, field string
		parameters              []symbol.TypeParam
		ctx                     Ctx
	}{
		{name: "struct_named_byte", structName: "byte", field: "byte"},
		{name: "parameter_named_int", structName: "Probe", field: "int", parameters: []symbol.TypeParam{{Name: "int"}}},
		{name: "other_source_type", structName: "Probe", field: "string", ctx: Ctx{currentFile: &symbol.FileScope{TopLevelClasses: []*symbol.ClassScope{{Class: &symbol.Definition{Name: "string", OriginalName: "String"}}}}}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fields := &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent(test.field)}}}
			result := fieldsWithAllocationIdentity(test.structName, fields, test.parameters, test.ctx)
			if len(result.List) != 2 {
				t.Fatalf("shadowed type %s incorrectly treated as nonzero", test.field)
			}
			if _, ok := result.List[1].Type.(*ast.StarExpr); !ok {
				t.Fatalf("marker depends on a named type: %#v", result.List[1].Type)
			}
		})
	}
}
