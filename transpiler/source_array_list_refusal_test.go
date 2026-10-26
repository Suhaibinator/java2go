package transpiler

import (
	"go/ast"
	"reflect"
	"testing"
)

func TestSourceArrayListP1StrictRefusalBeforeAST(t *testing.T) {
	for _, row := range []struct{ name, source, target string }{
		{"child_get", `class Parent extends java.util.ArrayList<Object>{} class Child extends Parent{public Object get(int n){return null;}}`, "Child"},
		{"child_size", `class Parent extends java.util.ArrayList<Object>{} class Child extends Parent{public int size(){return 1;}}`, "Child"},
		{"child_iterator", `class Parent extends java.util.ArrayList<Object>{} class Child extends Parent{public java.util.Iterator<Object> iterator(){return null;}}`, "Child"},
		{"direct_clear", `class Values extends java.util.ArrayList<Object>{public void clear(){}}`, "Values"},
		{"indexed_add", `class Values extends java.util.ArrayList<Object>{public void add(int n,Object value){}}`, "Values"},
		{"abstract_get", `abstract class Values extends java.util.ArrayList<Object>{public abstract Object get(int n);}`, "Values"},
		{"capacity_constructor", `class Values extends java.util.ArrayList<Object>{Values(){super(3);}}`, "Values"},
		{"generic_E_held", `class Values<E> extends java.util.ArrayList<E>{}`, "Values"},
		{"specialized_String_held", `class Values extends java.util.ArrayList<String>{}`, "Values"},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			scope := h.Ctx.currentFile.FindClassScope(row.target)
			ctx := classScopeCtx(scope, h.Ctx)
			ctx.className = scope.Class.Name
			strictRoutingState(t)
			ctx.importAliases = map[string]string{"kept": "kept"}
			ctx.usedImports = map[string]bool{"kept": true}
			var hoisted []ast.Decl
			ctx.hoistedDecls = &hoisted
			anonymous, labels := 7, 11
			ctx.anonClassCounter = &anonymous
			ctx.controlLabelCounter = &labels
			var result []ast.Decl
			var observed any
			func() {
				defer func() { observed = recover() }()
				result = parseDecls(scope.Class.DeclarationNode, h.File.Source, ctx)
			}()
			failure, ok := observed.(strictModeError)
			if !ok || failure.diagnostic.Kind != "ArrayList source capability" {
				t.Fatalf("unsupported canonical source declaration emitted AST instead of strict refusal: panic=%T %v declarations=%d", observed, observed, len(result))
			}
			if result != nil || len(hoisted) != 0 || anonymous != 7 || labels != 11 || !reflect.DeepEqual(ctx.importAliases, map[string]string{"kept": "kept"}) || !reflect.DeepEqual(ctx.usedImports, map[string]bool{"kept": true}) {
				t.Fatalf("refusal allocated partial output: result=%#v imports=%v used=%v hoisted=%d anonymous=%d labels=%d", result, ctx.importAliases, ctx.usedImports, len(hoisted), anonymous, labels)
			}
			if items := Diagnostics(); len(items) != 1 || items[0] != failure.diagnostic {
				t.Fatalf("strict refusal diagnostics = %#v", items)
			}
		})
	}
}

func TestSourceArrayListP1StrictProgramReturnsNoPartialAST(t *testing.T) {
	h := setupParseHelper(t, `class Parent extends java.util.ArrayList<Object>{} class Child extends Parent{public int size(){return 1;}}`)
	strictRoutingState(t)
	var result any
	var observed any
	func() { defer func() { observed = recover() }(); result = ParseNode(h.File.Ast, h.File.Source, h.Ctx) }()
	failure, ok := observed.(strictModeError)
	if !ok || failure.diagnostic.Kind != "ArrayList source capability" || result != nil {
		t.Fatalf("strict full-program API returned partial AST for unsupported descendant: panic=%T %v result=%T", observed, observed, result)
	}
}
