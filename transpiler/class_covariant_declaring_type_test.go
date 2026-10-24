package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrdinaryCovariantOverrideUsesDeclaringReturnType(t *testing.T) {
	assertOrdinaryCovariantDeclaringDescriptors(t, false)
}

func TestOrdinaryCovariantOverrideUsesDeclaringParameterType(t *testing.T) {
	assertOrdinaryCovariantDeclaringDescriptors(t, true)
}

func assertOrdinaryCovariantDeclaringDescriptors(t *testing.T, withParameter bool) {
	t.Helper()
	root := t.TempDir()
	for name, source := range map[string]string{
		"probe/api/Ancestor.java": `package probe.api; public class Ancestor { public String tag(){return "ancestor";} }
`,
		"probe/api/Factory.java": `package probe.api; public class Factory { public Ancestor value(){return new Ancestor();} }
`,
		"probe/impl/Descendant.java": `package probe.impl; import probe.api.Ancestor; public class Descendant extends Ancestor { public String tag(){return "descendant";} }
`,
		"probe/impl/SpecialFactory.java": `package probe.impl; import probe.api.Factory;
public class SpecialFactory extends Factory {
 public int calls; public Thread seen; public boolean nil; public boolean throwing;
 public static IllegalStateException marker=new IllegalStateException("marker");
 private Descendant result=new Descendant();
 public Descendant value(){ calls++; seen=Thread.currentThread(); if(throwing)throw marker; return nil?null:result; }
}
`,
	} {
		if withParameter && name == "probe/api/Factory.java" {
			source = strings.Replace(source, "public Ancestor value()", "public Ancestor copy(Ancestor input){return input;} public Ancestor value()", 1)
		}
		if withParameter && name == "probe/impl/SpecialFactory.java" {
			source = strings.Replace(source, "public Descendant value()", "public Descendant copy(probe.api.Ancestor input){return result;} public Descendant value()", 1)
		}
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	outputs := convertJavaProjectDir(t, root)
	parsed, err := parser.ParseFile(token.NewFileSet(), "SpecialFactory.go", outputs["probe/impl/SpecialFactory.go"], 0)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"Value": "Ancestor", "ValueJava2goExecution": "Ancestor", "ValueJava2goExactExecution": "Descendant"}
	if withParameter {
		want["Copy"] = "Ancestor"
		want["CopyJava2goExecution"] = "Ancestor"
		want["CopyJava2goExactExecution"] = "Descendant"
	}
	found := map[string]string{}
	for _, decl := range parsed.Decls {
		method, ok := decl.(*ast.FuncDecl)
		if !ok || method.Recv == nil || method.Type.Results == nil || len(method.Type.Results.List) != 1 {
			continue
		}
		if _, required := want[method.Name.Name]; !required {
			continue
		}
		if method.Name.Name == "CopyJava2goExecution" {
			last := method.Type.Params.List[len(method.Type.Params.List)-1].Type
			pointer, ok := last.(*ast.StarExpr)
			if !ok {
				t.Fatal("bridge parameter is not a source reference")
			}
			selector, ok := pointer.X.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Ancestor" {
				t.Fatalf("bridge parameter lost its declaring source package: %#v", last)
			}
		}
		pointer, ok := method.Type.Results.List[0].Type.(*ast.StarExpr)
		if !ok {
			t.Fatalf("%s result is not a source reference", method.Name.Name)
		}
		switch result := pointer.X.(type) {
		case *ast.Ident:
			found[method.Name.Name] = result.Name
		case *ast.SelectorExpr:
			found[method.Name.Name] = result.Sel.Name
		}
	}
	for name, result := range want {
		if found[name] != result {
			t.Fatalf("%s result=%q, want declaring-descriptor %q; generated:\n%s", name, found[name], result, outputs["probe/impl/SpecialFactory.go"])
		}
	}
}
