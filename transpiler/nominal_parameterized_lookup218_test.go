package transpiler

import (
	"encoding/json"
	"go/ast"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func nominalLookup218Basic(t *testing.T) (Ctx, *symbol.ClassScope, *symbol.ClassScope) {
	t.Helper()
	h := setupParseHelper(t, `package p;import java.util.function.Function;class Main{static void body(){class Local<U>{}}}class Outer<T>{class Inner<U>{}}`)
	ctx := classScopeCtx(h.File.Symbols.FindClassScope("Main"), h.Ctx)
	outer := h.File.Symbols.FindClassScope("Outer")
	if ctx.currentClass == nil || outer == nil || len(outer.Subclasses) != 1 {
		t.Fatal("actual basic nominal declarations missing")
	}
	return ctx, outer, outer.Subclasses[0]
}

func nominalLookup218Local(t *testing.T) (Ctx, *symbol.ClassScope) {
	t.Helper()
	ctx, _, _ := nominalLookup218Basic(t)
	ctx.localScope = ctx.currentClass.FindMethodByName("body", nil)
	if ctx.localScope == nil {
		t.Fatal("actual local declaring executable missing")
	}
	node := findNode(ctx.localScope.DeclarationNode, "class_declaration")
	if node == nil {
		t.Fatal("actual local declaration missing")
	}
	var declarations []ast.Decl
	counter := 0
	ctx.className = ctx.currentClass.Class.Name
	ctx.hoistedDecls, ctx.anonClassCounter = &declarations, &counter
	ctx.localClasses = map[string]*localClassInfo{}
	ctx.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{}
	ctx.executionContextName = "execution"
	hoistLocalClass(node, ctx.currentFile.Source, ctx)
	local := ctx.localClasses["Local"]
	if local == nil || local.scope == nil || !local.scope.Class.DeclarationNode.Equal(node) || localClassInDeclaration("Local", ctx) != local {
		t.Fatal("actual hoisted local scope/visibility missing")
	}
	return ctx, local.scope
}

func nominalLookup218Imported(t *testing.T) (Ctx, *symbol.ClassScope) {
	t.Helper()
	rawCompilationUnitGlobals(t)
	files := []parsing.SourceFile{
		{Name: "p/Main.java", Source: []byte(`package p;import q.Outer;class Main{}`)},
		{Name: "q/Outer.java", Source: []byte(`package q;public class Outer<T>{public class Inner<U>{}}`)},
		{Name: "p/Outer.java", Source: []byte(`package p;class Outer<T>{class Wrong{}}`)},
	}
	for i := range files {
		if err := files[i].ParseAST(); err != nil {
			t.Fatal("actual imported project AST setup failed")
		}
		symbol.AddSymbolsToPackage(files[i].ParseSymbols())
	}
	ResolveFiles(files)
	ctx := Ctx{currentFile: files[0].Symbols, currentClass: files[0].Symbols.FindClassScope("Main")}
	outer := files[1].Symbols.FindClassScope("Outer")
	if ctx.currentClass == nil || outer == nil || len(outer.Subclasses) != 1 || ctx.currentFile.Imports["Outer"] != "q" {
		t.Fatal("actual single-import source owner missing")
	}
	return ctx, outer.Subclasses[0]
}

func TestCI218NominalGenericLookupCompleteSegments(t *testing.T) {
	rows := []struct{ name, query, target string }{
		{"source_simple", "Outer<String>", "outer"},
		{"source_qualified_argument", "Outer<java.lang.String>", "outer"},
		{"nested_member", "Outer<java.lang.String>.Inner<java.lang.Integer>", "inner"},
		{"package_nested_member", "p.Outer<java.lang.String>.Inner<java.lang.Integer>", "inner"},
		{"external_function", "Function<java.lang.String,java.lang.String>", "external"},
		{"local_generic", "Local<java.lang.String>", "local"},
		{"single_import_generic", "Outer<java.lang.String>.Inner<java.lang.Integer>", "imported"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			ctx, outer, inner := nominalLookup218Basic(t)
			var want *symbol.ClassScope
			switch row.target {
			case "outer":
				want = outer
			case "inner":
				want = inner
			case "local":
				ctx, want = nominalLookup218Local(t)
			case "imported":
				ctx, want = nominalLookup218Imported(t)
			}
			var got *symbol.ClassScope
			output := qualifiedFunction218Logs(t, func() { got = resolveClassScopeByQualifiedName(ctx, row.query) })
			if got != want || len(output) != 0 {
				t.Fatal("complete parameterized nominal lookup lost owner or emitted truncated diagnostic")
			}
		})
	}
}

func TestCI218NominalLookupNamespaceShields(t *testing.T) {
	t.Run("binder_over_package", func(t *testing.T) {
		h := setupParseHelper(t, `package p;class Main<p>{}class Outer<T>{}`)
		ctx := classScopeCtx(h.File.Symbols.FindClassScope("Main"), h.Ctx)
		output := qualifiedFunction218Logs(t, func() {
			if resolveClassScopeByQualifiedName(ctx, "p.Outer") != nil {
				t.Fatal("captured binder reinterpreted as source package")
			}
		})
		if len(output) != 0 {
			t.Fatal("plain binder shield emitted a diagnostic")
		}
	})
	t.Run("actual_local", func(t *testing.T) {
		ctx, want := nominalLookup218Local(t)
		if resolveClassScopeByQualifiedName(ctx, "Local") != want {
			t.Fatal("actual local type visibility lost")
		}
	})
	t.Run("single_import_precedence", func(t *testing.T) {
		ctx, want := nominalLookup218Imported(t)
		if resolveClassScopeByQualifiedName(ctx, "Outer.Inner") != want {
			t.Fatal("single import lost to different same-package declaration")
		}
	})
}

func TestCI218NominalMalformedCompleteDiagnostic(t *testing.T) {
	rows := []struct{ name, query, message string }{
		{"unclosed", "Outer<java.lang.String", "Unbalanced angle brackets in type string: unclosed '<'"},
		{"stray_close", "Outer<java.lang.String>>", "Unbalanced angle brackets in type string: too many '>'"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			ctx, _, _ := nominalLookup218Basic(t)
			output := qualifiedFunction218Logs(t, func() {
				if resolveClassScopeByQualifiedName(ctx, row.query) != nil {
					t.Fatal("malformed nominal query acquired a source owner")
				}
			})
			var entry map[string]any
			if json.Unmarshal(output, &entry) != nil || len(entry) != 3 || entry["level"] != "warning" || entry["msg"] != row.message || entry["typeStr"] != row.query {
				t.Fatal("malformed nominal query did not retain one complete-input diagnostic")
			}
		})
	}
}
