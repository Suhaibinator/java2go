package transpiler

import (
	"sort"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func sourceGenericViewDemandTestContext(t *testing.T, sources map[string]string) Ctx {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
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
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
		files = append(files, file)
	}
	for _, file := range files {
		ResolveFile(file)
	}
	return Ctx{currentFile: files[0].Symbols, currentClass: files[0].Symbols.BaseClass}
}

func TestSourceGenericViewDemandConstructorClosure(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{
		"p/Base.java":   `package p; public class Base<T> {public T value; public Base(T value){this.value=value;} public int base(){return 11;} public T echo(T input){return input;}}`,
		"q/Middle.java": `package q; public class Middle<U> extends p.Base<U> {public Middle(U value){super(value);}}`,
		"app/Leaf.java": `package app; public class Leaf extends q.Middle<String> {public Leaf(String value){super(value);}}`,
		"app/Main.java": `package app; public class Main {p.Base<?> cast(Object value){return (p.Base<?>)value;} p.Base<?>[] array;}`,
	})
	seed := findQualifiedSourceClass("p.Base")
	plan, err := planGenericFamily(seed, ctx)
	if err != nil {
		t.Fatalf("unchanged constructor/closure audit rejected original hierarchy: %v", err)
	}
	for _, name := range []string{"p.Base", "q.Middle", "app.Leaf"} {
		member := findQualifiedSourceClass(name)
		if _, ok := plan.members[member]; !ok {
			t.Fatalf("complete planner lost %s", name)
		}
		if canonicalGenericFamily(member, ctx) == nil {
			t.Errorf("source wildcard demand did not activate complete member %s", name)
		}
	}
}

func TestSourceGenericViewDemandLexicalIdentity(t *testing.T) {
	for _, test := range []struct {
		name, use, imports string
		additional         map[string]string
		wantP, wantQ       bool
	}{
		{name: "wildcard field", use: `Cell<?> value;`, wantP: true},
		{name: "raw field", use: `Cell value;`, wantP: true},
		{name: "wildcard return", use: `Cell<?> read(){return null;}`, wantP: true},
		{name: "raw formal", use: `void use(Cell value){}`, wantP: true},
		{name: "wildcard local", use: `void use(){Cell<?> value=null;}`, wantP: true},
		{name: "raw local", use: `void use(){Cell value=null;}`, wantP: true},
		{name: "wildcard cast", use: `Object use(Object value){return (Cell<?>)value;}`, wantP: true},
		{name: "raw cast", use: `Object use(Object value){return (Cell)value;}`, wantP: true},
		{name: "unchecked concrete cast", use: `Object use(Object value){return (Cell<String>)value;}`, wantP: true},
		{name: "array cast", use: `Object use(Object value){return (Cell<String>[])value;}`, wantP: true},
		{name: "wildcard array field", use: `Cell<?>[][] values;`, wantP: true},
		{name: "raw array formal", use: `void use(Cell[] values){}`, wantP: true},
		{name: "wildcard upper bound", use: `Cell<? extends Number> value;`, wantP: true},
		{name: "wildcard lower bound", use: `Cell<? super String> value;`, wantP: true},
		{name: "qualified source", use: `p.Cell<?> value;`, imports: `import q.Cell;`, wantP: true},
		{name: "single import", use: `Cell<?> value;`, imports: `import q.Cell;`, wantQ: true},
		{name: "on demand import", use: `Cell<?> value;`, imports: `import p.*;`, wantP: true},
		{name: "same package shadow", use: `Cell<?> value;`, imports: `import p.*;`, additional: map[string]string{"app/Cell.java": `package app; public class Cell<T>{T value;}`}},
		{name: "nested source shadow", use: `static class Cell {} Object use(Object value){return (Cell)value;}`},
		{name: "method binder shadow", use: `<Cell> Cell use(Object value){return (Cell)value;}`},
		{name: "constructor binder shadow", use: `<Cell> Use(Object value){Cell local=(Cell)value;}`},
		{name: "class binder shadow", use: ``, additional: map[string]string{"app/Use.java": `package app; import p.Cell; public class Use<Cell>{Cell use(Object value){return (Cell)value;}}`}},
		{name: "concrete instantiation only", use: `Cell<String> value; Object make(){return new Cell<String>();}`},
		{name: "diamond is not raw", use: `Object make(){return new Cell<>();}`},
		{name: "external same suffix", use: `foreign.Cell<?> value;`},
	} {
		t.Run(test.name, func(t *testing.T) {
			imports := test.imports
			if imports == "" {
				imports = `import p.Cell;`
			}
			sources := map[string]string{
				"p/Cell.java":  `package p; public class Cell<T>{T value;}`,
				"q/Cell.java":  `package q; public class Cell<T>{T value;}`,
				"app/Use.java": `package app; ` + imports + ` public class Use{` + test.use + `}`,
			}
			for name, source := range test.additional {
				sources[name] = source
			}
			ctx := sourceGenericViewDemandTestContext(t, sources)
			for _, expected := range []struct {
				name     string
				demanded bool
			}{{"p.Cell", test.wantP}, {"q.Cell", test.wantQ}} {
				member := findQualifiedSourceClass(expected.name)
				if _, err := planGenericFamily(member, ctx); err != nil {
					t.Fatalf("control itself fails preserved audit: %s: %v", expected.name, err)
				}
				if got := canonicalGenericFamily(member, ctx) != nil; got != expected.demanded {
					t.Errorf("%s admitted = %t, want %t", expected.name, got, expected.demanded)
				}
			}
			if test.name == "same package shadow" && canonicalGenericFamily(findQualifiedSourceClass("app.Cell"), ctx) == nil {
				t.Error("same-package wildcard did not activate its own declaration")
			}
		})
	}
}

func TestSourceGenericViewDemandPreservesCompleteAudit(t *testing.T) {
	for _, test := range []struct{ name, declaration, extra string }{
		{name: "runtime field", declaration: `class Cell<T>{java.util.List<T> values;}`},
		{name: "array slot", declaration: `class Cell<T>{T[] values;}`},
		{name: "constructor owner local", declaration: `class Cell<T>{T value; Cell(T value){this.value=value;T copy=this.value;}}`},
		{name: "specialized runtime descendant", declaration: `class Cell<T>{T value;}`, extra: `class Bad extends Cell<java.util.List<String>>{}`},
		{name: "local subclass", declaration: `class Cell<T>{T value;}`, extra: `class ChildUse{void use(){class Local extends Cell<String>{}}}`},
		{name: "unsupported source bound", declaration: `class Limit{} class Cell<T extends Limit>{T value;}`},
		{name: "intersection bound", declaration: `class Cell<T extends Number & java.io.Serializable>{T value;}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, test.declaration+test.extra+`class Use{Cell<?> value;}`)
			member := helper.File.Symbols.FindClassScope("Cell")
			if plan, err := planGenericFamily(member, helper.Ctx); plan != nil || err == nil {
				t.Fatalf("unchanged complete audit admitted unsupported control: plan=%v err=%v", plan, err)
			}
			if canonicalGenericFamily(member, helper.Ctx) != nil {
				t.Fatal("lexical demand bypassed complete planner rejection")
			}
		})
	}
}
