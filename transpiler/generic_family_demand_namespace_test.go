package transpiler

import (
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func TestGenericDemandSourceNamesKeepNamespace(t *testing.T) {
	for _, test := range []struct {
		name    string
		sources map[string]string
		want    []string
	}{
		{"runtime names declared in source", map[string]string{"a/Use.java": `package a;class String<T>{}class Object<T>{}class Use{String<?> s;Object o;}`}, []string{"a.String", "a.Object"}},
		{"inherited member", map[string]string{"a/Use.java": `package a;class Base{static class Cell<T>{}}class Use extends Base{Cell<?> c;}`}, []string{"a.Base.Cell"}},
		{"static imported member", map[string]string{"a/Use.java": `package a;import static p.Owner.Cell;class Use{Cell<?> c;}`, "p/Owner.java": `package p;public class Owner{public static class Cell<T>{}}`}, []string{"p.Owner.Cell"}},
		{"qualified inherited member alias", map[string]string{"a/Use.java": `package a;class Base{static class Cell<T>{}}class Alias extends Base{}class Use{Alias.Cell<?> c;}`}, []string{"a.Base.Cell"}},
		{"nongeneric member shadows imported generic", map[string]string{"a/Use.java": `package a;import p.Cell;class Base{static class Cell{}}class Use extends Base{Cell c;}`, "p/Cell.java": `package p;public class Cell<T>{}`}, nil},
		{"local shadow and qualified escape", map[string]string{"a/Use.java": `package a;import p.Cell;class Use{void f(){class Cell<T>{}Cell<?> hidden=null;p.Cell<?> actual=null;}}`, "p/Cell.java": `package p;public class Cell<T>{}`}, []string{"p.Cell"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := sourceGenericViewDemandTestContext(t, test.sources)
			got := sourceGenericViewDemandSeeds(ctx)
			if len(got) != len(test.want) {
				t.Fatalf("seeds=%v want=%v", got, test.want)
			}
			for _, name := range test.want {
				found := false
				for _, seed := range got {
					found = found || seed == findQualifiedSourceClass(name)
				}
				if !found {
					t.Errorf("missing %s", name)
				}
			}
		})
	}
}

func TestGenericDemandSourceNamesRefreshAfterMutation(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{"Use.java": `class Use{String s;}`})
	if got := sourceGenericViewDemandSeeds(ctx); len(got) != 0 {
		t.Fatal("runtime name unexpectedly source backed")
	}
	file := parsing.SourceFile{Name: "String.java", Source: []byte(`class String<T>{}`)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	file.ParseSymbols()
	symbol.AddSymbolsToPackage(file.Symbols)
	ResolveFile(file)
	target := file.Symbols.BaseClass
	check := func(want *symbol.ClassScope) {
		t.Helper()
		got := sourceGenericViewDemandSeeds(ctx)
		if want == nil {
			if len(got) != 0 {
				t.Fatal("stale source declaration")
			}
			return
		}
		if len(got) != 1 || got[0] != want {
			t.Fatal("fresh source declaration missing")
		}
	}
	check(target)
	pkg := symbol.GlobalScope.Packages[file.Symbols.Package]
	var key string
	for name, existing := range pkg.Files {
		if existing == file.Symbols {
			key = name
			break
		}
	}
	delete(pkg.Files, key)
	check(nil)
	pkg.Files[key] = file.Symbols
	check(target)
	prior := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	check(nil)
	symbol.GlobalScope = prior
	check(target)
}

func TestGenericDemandSourceNamesSyntheticBindings(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{"Use.java": `class Use{String s;}`})
	// Synthesized declarations can be visible through rendering registries while
	// remaining absent from the ordinary graph; preserve the original resolver.
	file := parsing.SourceFile{Name: "Synthetic.java", Source: []byte(`class Synthetic<T>{}`)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	file.ParseSymbols()
	target := file.Symbols.BaseClass
	for _, anonymous := range []bool{false, true} {
		active := ctx.Clone()
		if anonymous {
			active.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{{}: {structName: "String", scope: target}}
		} else {
			active.localClasses = map[string]*localClassInfo{"String": {scope: target}}
		}
		got := sourceGenericViewDemandSeeds(active)
		if len(got) != 1 || got[0] != target {
			t.Fatalf("anonymous=%t synthesized binding lost", anonymous)
		}
	}
}
