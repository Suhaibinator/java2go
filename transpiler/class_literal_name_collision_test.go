package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

func TestClassLiteralGeneratedPackageNamesRemainDistinct(t *testing.T) {
	outputs := convertJavaProjectDir(t, filepath.Join("testdata", "classliteral_name_collision94", "project", "src", "main", "java"))
	packages := make(map[string]map[string]string)
	for path, output := range outputs {
		file, err := parser.ParseFile(token.NewFileSet(), path, output, 0)
		if err != nil {
			t.Fatal(err)
		}
		owner := filepath.Dir(path)
		if packages[owner] == nil {
			packages[owner] = make(map[string]string)
		}
		add := func(name string) {
			if name == "_" || name == "init" {
				return
			}
			if previous, exists := packages[owner][name]; exists {
				t.Errorf("generated package %s redeclares %s in %s and %s", owner, name, previous, path)
			}
			packages[owner][name] = path
		}
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				if decl.Recv == nil {
					add(decl.Name.Name)
				}
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						add(spec.Name.Name)
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							add(name.Name)
						}
					}
				}
			}
		}
	}
}
