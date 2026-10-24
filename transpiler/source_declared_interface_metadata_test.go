package transpiler

import (
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"strconv"
	"testing"
)

func interface136RegistrationIDs(t *testing.T, declaration ast.Decl) map[string]int {
	t.Helper()
	ids := map[string]int{}
	calls := 0
	ast.Inspect(declaration, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "RegisterJavaType" {
			return true
		}
		calls++
		for _, arg := range call.Args[2:] {
			if id, ok := arg.(*ast.CallExpr); ok && len(id.Args) == 1 {
				if literal, ok := id.Args[0].(*ast.BasicLit); ok {
					v, e := strconv.Unquote(literal.Value)
					if e == nil {
						ids[v]++
					}
				}
			}
		}
		return false
	})
	if calls != 1 {
		t.Fatalf("registration calls=%d, want exactly one", calls)
	}
	return ids
}

func TestInterface136DeclarationOwnedExternalEdges(t *testing.T) {
	cases := []struct {
		name, source  string
		want          []string
		deny          []string
		alias, caller bool
	}{
		{name: "qualified binder prefix", source: `package p; class Value<java> implements java.lang.Comparable<Value<java>>{public int compareTo(Value<java> v){return 0;}}`, deny: []string{"java.lang.Comparable"}},
		{name: "unknown nested binary identity", source: `package p; class Value implements foreign.Outer.Comparable<Value>{public int compareTo(Value v){return 0;}}`, deny: []string{"java.lang.Comparable", "foreign.Outer.Comparable"}},
		{name: "explicit canonical precedes unknown demand", source: `package p; import unknown.*; import java.lang.Comparable; class Value implements Comparable<Value>{public int compareTo(Value v){return 0;}}`, want: []string{"java.lang.Comparable"}},
		{name: "qualified canonical precedes unknown demand", source: `package p; import unknown.*; class Value implements java.lang.Comparable<Value>{public int compareTo(Value v){return 0;}}`, deny: []string{"java.lang.Comparable"}},
		{name: "source class competition", source: `package p; class Comparable<T>{} class Value implements Comparable<Value>{public int compareTo(Value v){return 0;}}`, deny: []string{"java.lang.Comparable"}},
		{name: "demand real interface", source: `package p; import java.util.*; class Value implements Comparator<Value>{public int compare(Value a,Value b){return 0;}}`, want: []string{"java.util.Comparator"}},
		{name: "demand class is not interface", source: `package p; import java.util.*; class Value implements ArrayList{}`, deny: []string{"java.util.ArrayList"}},
		{name: "private demand member is not importable", source: `package p; import java.util.*; class Value implements Tripwire{}`, deny: []string{"java.util.Tripwire"}},
		{name: "qualified", source: `package p; class Value implements java.lang.Comparable<Value>{public int compareTo(Value v){return 0;}}`, want: []string{"java.lang.Comparable"}},
		{name: "implicit", source: `package p; class Value implements Comparable<Value>{public int compareTo(Value v){return 0;}}`, want: []string{"java.lang.Comparable"}},
		{name: "platform demand", source: `package p; import java.util.*; class Value implements Comparable<Value>{public int compareTo(Value v){return 0;}}`, want: []string{"java.lang.Comparable"}},
		{name: "raw", source: `package p; import java.lang.Comparable; class Value implements Comparable{public int compareTo(Object v){return 0;}}`, want: []string{"java.lang.Comparable"}},
		{name: "header body shadow", source: `package p; import java.lang.Comparable; class Value implements Comparable<Value>{static class Comparable{} public int compareTo(Value v){return 0;}}`, want: []string{"java.lang.Comparable"}},
		{name: "multiple", source: `package p; import java.io.Serializable; class Value implements Comparable<Value>,Serializable{public int compareTo(Value v){return 0;}}`, want: []string{"java.lang.Comparable", "java.io.Serializable"}},
		{name: "subinterface", source: `package p; interface Value<T> extends java.lang.Comparable<T>{}`, want: []string{"java.lang.Comparable"}},
		{name: "source interface", source: `package p; interface Comparable<T>{int compareTo(T v);} class Value implements Comparable<Value>{public int compareTo(Value v){return 0;}}`, want: []string{"p.Comparable"}, deny: []string{"java.lang.Comparable"}},
		{name: "default source", source: `interface Comparable<T>{int compareTo(T v);} class Value implements Comparable<Value>{public int compareTo(Value v){return 0;}}`, want: []string{"Comparable"}, deny: []string{"java.lang.Comparable"}},
		{name: "explicit foreign", source: `package p; import foreign.Comparable; class Value implements Comparable<Value>{public int compareTo(Value v){return 0;}}`, want: []string{"foreign.Comparable"}, deny: []string{"java.lang.Comparable"}},
		{name: "unknown demand", source: `package p; import unknown.*; class Value implements Comparable<Value>{public int compareTo(Value v){return 0;}}`, deny: []string{"java.lang.Comparable"}},
		{name: "array invalid declaration", source: `package p; class Value implements java.lang.Comparable[]{}`, deny: []string{"java.lang.Comparable"}},
		{name: "source binder invalid declaration", source: `package p; class Value<Comparable> implements Comparable{}`, deny: []string{"java.lang.Comparable"}},
		{name: "Go alias is not Java binder", source: `package p; class Value<T> implements Comparable<Value<T>>{public int compareTo(Value<T> v){return 0;}}`, want: []string{"java.lang.Comparable"}, alias: true},
		{name: "caller binder is not header", source: `package p; class Value implements Comparable<Value>{public int compareTo(Value v){return 0;}}`, want: []string{"java.lang.Comparable"}, caller: true},
		{name: "qualified source prefix", source: `package p; class java {static class lang{interface Comparable<T>{int compareTo(T v);}}} class Value implements java.lang.Comparable<Value>{public int compareTo(Value v){return 0;}}`, want: []string{"p.java$lang$Comparable"}, deny: []string{"java.lang.Comparable"}},
		{name: "missing source prefix member", source: `package p; class java{} class Value implements java.lang.Comparable<Value>{public int compareTo(Value v){return 0;}}`, deny: []string{"java.lang.Comparable"}},
	}
	for _, row := range cases {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			if row.name == "explicit foreign" {
				foreign := parsing.SourceFile{Name: "foreign/Comparable.java", Source: []byte(`package foreign; public interface Comparable<T>{int compareTo(T v);}`)}
				if e := foreign.ParseAST(); e != nil {
					t.Fatal(e)
				}
				foreign.ParseSymbols()
				symbol.AddSymbolsToPackage(foreign.Symbols)
			}
			ctx := h.Ctx.Clone()
			scope := h.File.Symbols.FindClassScope("Value")
			if scope == nil {
				t.Fatal("missing parsed Value")
			}
			ctx.currentClass = scope
			ctx.className = scope.Class.Name
			if row.alias {
				scope.TypeParameters[0].Declaration.GoName = "Comparable"
			}
			if row.caller {
				ctx.syntheticTypeParameters = []symbol.TypeParam{symbol.NewTypeParam("Comparable", nil)}
				ctx.localScope = &symbol.Definition{TypeParameters: ctx.syntheticTypeParameters}
			}
			var sourceIDs []ast.Expr
			for _, edge := range resolveImplementedInterfaceScopesInDeclaringContext(ctx, scope) {
				sourceIDs = append(sourceIDs, javaTypeIDLiteral(javaClassBinaryName(edge), ctx))
			}
			for kind, decl := range map[string]ast.Decl{"named": sourceClassRegistrationDecl(scope, ctx), "synthetic": syntheticReferenceRegistrationDecl(ctx.className, "p.synthetic.Value", nil, sourceIDs, ctx)} {
				ids := interface136RegistrationIDs(t, decl)
				for _, id := range row.want {
					if ids[id] != 1 {
						t.Errorf("%s edge %s count=%d; IDs=%v", kind, id, ids[id], ids)
					}
				}
				for _, id := range row.deny {
					if ids[id] != 0 {
						t.Errorf("%s stolen edge %s; IDs=%v", kind, id, ids)
					}
				}
			}
		})
	}
}

func TestInterface136SourceAndImportedDeclarationContext(t *testing.T) {
	old := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = old })
	var files []parsing.SourceFile
	for i, text := range []string{`package foreign; public interface Comparable<T>{int compareTo(T v);}`, `package p; import foreign.Comparable; public class Value implements Comparable<Value>{public int compareTo(Value v){return 0;}}`, `package caller; class Caller<Comparable>{}`} {
		f := parsing.SourceFile{Name: strconv.Itoa(i) + ".java", Source: []byte(text)}
		if e := f.ParseAST(); e != nil {
			t.Fatal(e)
		}
		f.ParseSymbols()
		symbol.AddSymbolsToPackage(f.Symbols)
		files = append(files, f)
	}
	value := files[1].Symbols.BaseClass
	ctx := Ctx{currentFile: files[2].Symbols, currentClass: files[2].Symbols.BaseClass, className: value.Class.Name}
	ids := interface136RegistrationIDs(t, sourceClassRegistrationDecl(value, ctx))
	if ids["foreign.Comparable"] != 1 || ids["java.lang.Comparable"] != 0 {
		t.Fatalf("declaration import lost or caller binder captured it: %v", ids)
	}
}

func TestInterface136UnanchoredDeclarationDeclines(t *testing.T) {
	h := setupParseHelper(t, `package p; class Value implements Comparable<Value>{public int compareTo(Value v){return 0;}}`)
	scope := h.File.Symbols.FindClassScope("Value")
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	if ids := sourceDeclaredExternalInterfaceIDs(scope, h.Ctx); len(ids) != 0 {
		t.Fatalf("unanchored declaration borrowed caller context: %v", ids)
	}
}
