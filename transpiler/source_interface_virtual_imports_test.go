package transpiler

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func virtualImportTestContext(t *testing.T, sources map[string]string, target string) Ctx {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	files := make([]parsing.SourceFile, 0, len(names))
	for _, name := range names {
		file := parsing.SourceFile{Name: name, Source: []byte(sources[name])}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		symbol.AddSymbolsToPackage(file.ParseSymbols())
		files = append(files, file)
	}
	for _, file := range files {
		ResolveFile(file)
	}
	for _, file := range files {
		if file.Name == target {
			return Ctx{currentFile: file.Symbols, currentClass: file.Symbols.BaseClass,
				importAliases: map[string]string{stdjavaImportPath: "runtimeABI", "reserved.pkg": "real"},
				usedImports:   map[string]bool{stdjavaImportPath: true, "reserved.pkg": true}}
		}
	}
	t.Fatal("actual target source missing")
	return Ctx{}
}

func virtualImportTestText(t *testing.T, value any) string {
	t.Helper()
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), value); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestSourceInterfaceVirtualRejectedPlanImports(t *testing.T) {
	root := filepath.Join("testdata", "method_reflection_inherited_origins94", "project", "src", "main", "java")
	sources := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(path) != ".java" {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sources[filepath.ToSlash(rel)] = string(data)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(sources) != 6 {
		t.Fatalf("original source count=%d", len(sources))
	}
	ctx := virtualImportTestContext(t, sources, "origin/leaf/Child.java")
	iface := findQualifiedSourceClass("origin.api.Producer")
	base := findQualifiedSourceClass("origin.base.Base")
	if iface == nil || base == nil || len(base.TypeParameters) != 1 {
		t.Fatal("actual interface/generic superclass declarations missing")
	}
	method := iface.FindMethod().ByOriginalName("echo")[0]
	proof := ctx.Clone()
	proof.importAliases, proof.usedImports = maps.Clone(ctx.importAliases), maps.Clone(ctx.usedImports)
	plan := planSourceInterfaceVirtualEntry(iface, method, proof)
	if plan == nil || !proof.usedImports["origin.real"] {
		t.Fatal("real declaration signature did not exercise package mapping")
	}
	if sourceInterfaceVirtualImplementation(ctx.currentClass, plan, proof) != nil {
		t.Fatal("generic inherited owner unexpectedly acquired a virtual entry")
	}
	field := ctx.currentClass.FindFieldByName("unrelated")
	if field == nil || qualifyDeclaredReferenceType(symbol.JavaType{Original: field.OriginalType}, ctx) != "origin.shadow.Token" {
		t.Fatal("child field lost its actual shadow declaration")
	}
	t.Logf("interface=%s owner=%s result=%s signature=%s proofImports=%v",
		qualifiedSourceClassName(iface), qualifiedSourceClassName(base), method.OriginalType,
		virtualImportTestText(t, plan.signature), proof.usedImports)
	aliases, imports := maps.Clone(ctx.importAliases), maps.Clone(ctx.usedImports)
	if declarations := generateSourceInterfaceVirtualEntries(ctx); len(declarations) != 0 {
		t.Fatalf("discarded candidate emitted %d entries", len(declarations))
	}
	if !reflect.DeepEqual(ctx.importAliases, aliases) || !reflect.DeepEqual(ctx.usedImports, imports) {
		t.Fatalf("discarded physical signature published imports: aliases=%v used=%v; before=%v %v", ctx.importAliases, ctx.usedImports, aliases, imports)
	}
}

func TestSourceInterfaceVirtualMixedPlanImports(t *testing.T) {
	for _, order := range []string{"api.Producer, api.Counter", "api.Counter, api.Producer"} {
		t.Run(order, func(t *testing.T) {
			ctx := virtualImportTestContext(t, map[string]string{
				"api/Producer.java": `package api;public interface Producer{real.Token echo(String value);}`,
				"api/Counter.java":  `package api;public interface Counter{int code();}`,
				"real/Token.java":   `package real;public class Token{}`,
				"base/Base.java":    `package base;public class Base<T>{public real.Token echo(T value){return null;}}`,
				"leaf/Child.java":   `package leaf;public class Child extends base.Base<String> implements ` + order + `{public int code(){return 7;}}`,
				"leaf/Grand.java":   `package leaf;public class Grand extends Child{public int code(){return 9;}}`,
			}, "leaf/Child.java")
			declarations := generateSourceInterfaceVirtualEntries(ctx)
			if len(declarations) != 1 || !strings.Contains(virtualImportTestText(t, declarations[0]), "int32") {
				t.Fatalf("genuine second entry missing: %v", declarations)
			}
			if ctx.usedImports["real"] || ctx.importAliases["real"] != "" {
				t.Fatalf("rejected candidate polluted accepted candidate: %v %v", ctx.importAliases, ctx.usedImports)
			}
			if ctx.importAliases["reserved.pkg"] != "real" || !ctx.usedImports["reserved.pkg"] {
				t.Fatal("preexisting imports changed")
			}
		})
	}
}

func TestSourceInterfaceVirtualRequiredPlanImports(t *testing.T) {
	ctx := virtualImportTestContext(t, map[string]string{
		"api/View.java":    `package api;public interface View{real.Token echo(real.Token real);}`,
		"real/Token.java":  `package real;public class Token{}`,
		"base/Parent.java": `package base;public class Parent implements api.View{public real.Token echo(real.Token real){return real;}}`,
		"leaf/Token.java":  `package leaf;public class Token{}`,
		"leaf/Child.java":  `package leaf;public class Child extends base.Parent{public real.Token echo(real.Token real){return real;}}`,
	}, "leaf/Child.java")
	declarations := generateSourceInterfaceVirtualEntries(ctx)
	if len(declarations) != 1 {
		t.Fatalf("required entry count=%d", len(declarations))
	}
	alias := ctx.importAliases["real"]
	text := virtualImportTestText(t, declarations[0])
	if alias == "" || alias == "real" || !ctx.usedImports["real"] || !strings.Contains(text, "*"+alias+".Token") {
		t.Fatalf("required declaration import/alias lost: %v %v\n%s", ctx.importAliases, ctx.usedImports, text)
	}
	decl := declarations[0].(*ast.FuncDecl)
	if decl.Type.Params.List[0].Names[0].Name == "real" || !strings.Contains(text, "runtimeABI.Execution") {
		t.Fatal("caller execution/argument declaration changed")
	}
}

func TestSourceInterfaceVirtualCovariantProofImports(t *testing.T) {
	ctx := virtualImportTestContext(t, map[string]string{
		"api/View.java":    `package api;public interface View{Object value();}`,
		"real/Token.java":  `package real;public class Token{}`,
		"base/Parent.java": `package base;public class Parent implements api.View{public real.Token value(){return null;}}`,
		"leaf/Child.java":  `package leaf;public class Child extends base.Parent{public real.Token value(){return null;}}`,
	}, "leaf/Child.java")
	declarations := generateSourceInterfaceVirtualEntries(ctx)
	if len(declarations) != 1 {
		t.Fatalf("covariant entry count=%d", len(declarations))
	}
	text := virtualImportTestText(t, declarations[0])
	if !strings.Contains(text, "any") || strings.Contains(text, ".Token") {
		t.Fatalf("proof-only result unexpectedly emitted:\n%s", text)
	}
	if ctx.usedImports["real"] || ctx.importAliases["real"] != "" {
		t.Fatalf("accepted wrapper published discarded proof result: %v %v", ctx.importAliases, ctx.usedImports)
	}
}
