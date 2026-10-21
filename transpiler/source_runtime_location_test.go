package transpiler

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceRuntimeLocationWithoutReflection(t *testing.T) {
	rawCompilationUnitGlobals(t)
	root := t.TempDir()
	input := filepath.Join(root, "SourceControl.java")
	output := filepath.Join(root, "generated")
	source := `package places;
 interface Marker{} class Parent{}
 class SourceControl<T> extends Parent implements Marker {
 static class Nested{}
 Object local(){class Local{} return new Local();}
 Object anonymous(){return new Object(){};}
 }
 enum Pick{ONE{int tag(){return 1;}},TWO;int tag(){return 2;}}
 `
	if err := os.WriteFile(input, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	var diagnostics bytes.Buffer
	if err := runInternal([]string{"-strict", "-sync", "-w", "-output", output, input}, &diagnostics, false); err != nil {
		t.Fatalf("strict source producer: %v; %s", err, diagnostics.Bytes())
	}
	if sourceUsesReflection() {
		t.Fatal("fixture must not activate reflection-demand metadata")
	}
	sourceIDs, locationIDs := map[string]int{}, map[string]int{}
	if err := filepath.WalkDir(output, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".go" {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			var buffer bytes.Buffer
			if err := format.Node(&buffer, token.NewFileSet(), call.Args[0]); err != nil {
				t.Fatal(err)
			}
			switch selector.Sel.Name {
			case "RegisterJavaSourceType":
				sourceIDs[buffer.String()]++
			case "RegisterClassRuntimeLocation":
				locationIDs[buffer.String()]++
			}
			return true
		})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(sourceIDs) < 7 {
		t.Fatalf("named/interface/nested/local/anonymous/enum source inventory incomplete: %v", sourceIDs)
	}
	for id, count := range sourceIDs {
		if count != 1 || locationIDs[id] != 1 {
			t.Errorf("source declaration %s ownership=%d location=%d; want both exactly once without reflection", id, count, locationIDs[id])
		}
	}
	if len(locationIDs) != len(sourceIDs) {
		t.Errorf("location declarations=%d source declarations=%d", len(locationIDs), len(sourceIDs))
	}
}

func TestSourceRuntimeLocationExplicitProfileAndCopies(t *testing.T) {
	h := setupParseHelper(t, `package locations; class Value{static class Child{}}`)
	for _, ctx := range []Ctx{h.Ctx, h.Ctx.Clone()} {
		if stmt := sourceRuntimeLocationRegistrationStmt(javaTypeIDLiteral("locations.Value", ctx), ctx); stmt != nil {
			t.Fatal("default source-owned context invented app provenance")
		}
	}
	ctx := h.Ctx
	ctx.projectMode = true
	if sourceRuntimeLocationRegistrationStmt(javaTypeIDLiteral("locations.Value", ctx), ctx) != nil {
		t.Fatal("project-mode flag alone invented app provenance")
	}
	ctx.runtimeLocationProfile = sourceRuntimeLocationClasspath
	for _, copy := range []Ctx{ctx, ctx.Clone(), classScopeCtx(ctx.currentClass, ctx)} {
		stmt := sourceRuntimeLocationRegistrationStmt(javaTypeIDLiteral("locations.Value", copy), copy)
		if stmt == nil {
			t.Fatal("explicit profile lost across child context")
		}
		var out bytes.Buffer
		if err := format.Node(&out, token.NewFileSet(), stmt); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"RegisterClassRuntimeLocation", "Known: true", `ModuleName: ""`, "LoaderIdentity: stdjava.AppClassLoaderIdentity", `LoaderName: "app"`} {
			if !strings.Contains(out.String(), want) {
				t.Errorf("explicit declaration lacks %q: %s", want, out.String())
			}
		}
	}
}
