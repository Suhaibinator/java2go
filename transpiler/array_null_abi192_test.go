package transpiler

import (
	_ "embed"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

//go:embed testdata/array-null-abi192/ArrayNullABI192.java
var arrayNullABI192Java string
var arrayNullABI192Methods = []struct {
	name   string
	helper bool
}{
	{"pObject", false}, {"pPrimitive", false}, {"pPrimitiveReverse", false}, {"pNestedPrimitive", false}, {"pNestedReference", false}, {"pGenericReference", false}, {"pNullableAlias", false}, {"pGenericArray", false}, {"pParenthesized", false}, {"pReverseReference", false},
	{"nObject", true}, {"nScalar", true}, {"nWildcard", true}, {"nErasedField", true}, {"nCall", true}, {"nExplicitCast", true}, {"nField", true}, {"nString", true}, {"nInterface", true}, {"nArrayElement", true}}

func arrayNullABI192Return(t *testing.T, file *ast.File, name string) ast.Expr {
	t.Helper()
	for _, decl := range file.Decls {
		method, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasSuffix(method.Name.Name, "Java2goExecution") || !strings.EqualFold(strings.TrimSuffix(method.Name.Name, "Java2goExecution"), name) {
			continue
		}
		for _, s := range method.Body.List {
			if r, ok := s.(*ast.ReturnStmt); ok && len(r.Results) == 1 {
				return r.Results[0]
			}
		}
	}
	t.Fatalf("missing compared return for%s", name)
	return nil
}
func arrayNullABI192Helper(expr ast.Expr) bool {
	for {
		if p, ok := expr.(*ast.ParenExpr); ok {
			expr = p.X
			continue
		}
		if n, ok := expr.(*ast.UnaryExpr); ok && n.Op == token.NOT {
			expr = n.X
			continue
		}
		break
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "JavaReferenceEqual"
}
func TestArrayNullABI192WorkBudget(t *testing.T) {
	generated := renderGoFileFromJava(t, arrayNullABI192Java)
	if dir := os.Getenv("ARRAY_NULL192_ARTIFACT"); dir != "" {
		if e := os.WriteFile(filepath.Join(dir, "work-generated.go"), []byte(generated), 0600); e != nil {
			t.Fatal(e)
		}
	}
	file, e := parser.ParseFile(token.NewFileSet(), "generated.go", generated, 0)
	if e != nil {
		t.Fatal(e)
	}
	for _, row := range arrayNullABI192Methods {
		if row.helper {
			continue
		}
		t.Run(row.name, func(t *testing.T) {
			has := arrayNullABI192Helper(arrayNullABI192Return(t, file, row.name))
			if has {
				t.Fatal("canonical array-null helper entries=1 maximum=0")
			}
		})
	}
}
func TestArrayNullABI192ErasedControls(t *testing.T) {
	generated := renderGoFileFromJava(t, arrayNullABI192Java)
	file, e := parser.ParseFile(token.NewFileSet(), "generated.go", generated, 0)
	if e != nil {
		t.Fatal(e)
	}
	for _, row := range arrayNullABI192Methods {
		if !row.helper {
			continue
		}
		t.Run(row.name, func(t *testing.T) {
			if !arrayNullABI192Helper(arrayNullABI192Return(t, file, row.name)) {
				t.Fatal("unproved erased/null guard lost identity helper")
			}
		})
	}
}
