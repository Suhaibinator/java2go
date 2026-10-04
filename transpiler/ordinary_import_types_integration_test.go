package transpiler

import (
	"context"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func ordinaryDemandTestContext(t *testing.T, sources map[string]string, caller string) Ctx {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	diagnostics.mu.Lock()
	previousItems, previousStrict := diagnostics.items, diagnostics.strict
	diagnostics.items, diagnostics.strict = nil, false
	diagnostics.mu.Unlock()
	t.Cleanup(func() {
		symbol.GlobalScope = previous
		diagnostics.mu.Lock()
		diagnostics.items, diagnostics.strict = previousItems, previousStrict
		diagnostics.mu.Unlock()
	})
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	var current *symbol.FileScope
	for _, name := range names {
		file := parsing.SourceFile{Name: name, Source: []byte(sources[name])}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		scope := file.ParseSymbols()
		symbol.AddSymbolsToPackage(scope)
		if name == caller {
			current = scope
		}
	}
	if current == nil {
		t.Fatal("missing caller source")
	}
	return Ctx{currentFile: current, currentClass: current.BaseClass}
}

func TestOrdinaryDemandTypesRespectScopeAndAccessibility(t *testing.T) {
	tests := []struct {
		name, imports, query, expected string
		extras                         map[string]string
		foreignContext                 bool
	}{
		{name: "package constructor type", imports: "import p.*;", query: "Value", expected: "p.Value"},
		{name: "duplicate import identity", imports: "import p.*; import p.*;", query: "Value", expected: "p.Value"},
		{name: "inaccessible competitor", imports: "import p.*; import q.*;", query: "Value", expected: "p.Value", extras: map[string]string{"q/Value.java": "package q; class Value {}"}},
		{name: "same package precedence", imports: "import p.*;", query: "Value", expected: "app.Value", extras: map[string]string{"app/Value.java": "package app; public class Value {}"}},
		{name: "single import precedence", imports: "import p.*; import q.Value;", query: "Value", expected: "q.Value", extras: map[string]string{"q/Value.java": "package q; public class Value {}"}},
		{name: "nonstatic member demand", imports: "import p.Outer.*;", query: "Inner", expected: "p.Outer.Inner"},
		{name: "ordinary member demand excludes inherited member", imports: "import p.PublicOwner.*;", query: "Member", expected: ""},
		{name: "interface implicit public member", imports: "import p.Contract.*;", query: "Token", expected: "p.Contract.Token"},
		{name: "private member excluded", imports: "import p.Outer.*;", query: "Secret", expected: ""},
		{name: "package demand excludes unrelated nested", imports: "import p.*;", query: "Inner", expected: ""},
		{name: "caller file owns import tree", imports: "import p.*;", query: "Value", expected: "p.Value", foreignContext: true},
		{name: "explicit source String disambiguates", imports: "import p.String; import p.*;", query: "String", expected: "p.String", extras: map[string]string{"p/String.java": "package p; public class String {}"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sources := map[string]string{
				"p/Value.java":       "package p; public class Value {}",
				"p/Outer.java":       "package p; public class Outer {public class Inner {} private class Secret {}}",
				"p/PublicOwner.java": "package p; class HiddenBase {public static class Member {}} public class PublicOwner extends HiddenBase {}",
				"p/Contract.java":    "package p; public interface Contract {class Token {}}",
				"foreign/Other.java": "package foreign; import q.*; public class Other {}",
				"app/Main.java":      "package app; " + test.imports + " public class Main {}",
			}
			for name, source := range test.extras {
				sources[name] = source
			}
			ctx := ordinaryDemandTestContext(t, sources, "app/Main.java")
			if test.foreignContext {
				ctx.currentClass = findQualifiedSourceClass("foreign.Other")
			}
			actual := qualifiedSourceClassName(resolveClassScopeByQualifiedName(ctx, test.query))
			if actual != test.expected {
				t.Fatalf("type %s bound to %q, want declaration %q", test.query, actual, test.expected)
			}
			if len(collectedDiagnostics()) != 0 {
				t.Fatalf("valid resolution diagnosed: %v", collectedDiagnostics())
			}
		})
	}
}

func ordinaryDemandAssertJavacAmbiguity(t *testing.T, sources map[string]string, name string) {
	t.Helper()
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	paths := make([]string, 0, len(sources))
	for name, source := range sources {
		path := filepath.Join(root, name)
		if err := writeProjectFile(path, []byte(source)); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, javac, append([]string{"-d", filepath.Join(root, "classes")}, paths...)...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatal(ctx.Err())
	}
	if err == nil || !strings.Contains(string(output), "reference to "+name+" is ambiguous") {
		t.Fatalf("fresh JVM did not reject ambiguous type %s: %v\n%s", name, err, output)
	}
	t.Logf("fresh javac confirmed ambiguous type %s", name)
}

func TestOrdinaryDemandTypesDiagnoseDistinctAccessibleCandidates(t *testing.T) {
	tests := []struct {
		name, imports, query string
		sources              map[string]string
	}{
		{"two packages", "import p.*; import q.*;", "Choice", map[string]string{"p/Choice.java": "package p; public class Choice {}", "q/Choice.java": "package q; public class Choice {}"}},
		{"ordinary and static", "import p.*; import static q.Owner.*;", "Choice", map[string]string{"p/Choice.java": "package p; public class Choice {}", "q/Owner.java": "package q; public class Owner {public static class Choice {}}"}},
		{"implicit java.lang", "import p.*;", "String", map[string]string{"p/String.java": "package p; public class String {}"}},
		{"known external demand", "import p.*; import java.util.*;", "List", map[string]string{"p/List.java": "package p; public class List {}"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sources := map[string]string{}
			for name, source := range test.sources {
				sources[name] = source
			}
			sources["app/Main.java"] = "package app; " + test.imports + " public class Main {" + test.query + " value;}"
			ordinaryDemandAssertJavacAmbiguity(t, sources, test.query)
			ctx := ordinaryDemandTestContext(t, sources, "app/Main.java")
			if scope := resolveClassScopeByQualifiedName(ctx, test.query); scope != nil {
				t.Fatalf("ambiguous type selected declaration %s", qualifiedSourceClassName(scope))
			}
			found := false
			for _, diagnostic := range collectedDiagnostics() {
				found = found || strings.Contains(diagnostic.Kind, "ambiguous imported type "+test.query)
			}
			if !found {
				t.Fatalf("missing ambiguity diagnostic for %s: %v", test.query, collectedDiagnostics())
			}
		})
	}
}

func TestOrdinaryDemandConstructorsFieldsFormalsReturnsAndExtendsProjectJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                        `<project><modelVersion>4.0.0</modelVersion><groupId>demand</groupId><artifactId>types</artifactId><version>1</version></project>`,
		"src/main/java/p/Value.java":     `package p; public class Value {public int read(){return 7;}}`,
		"src/main/java/p/Base.java":      `package p; public class Base {public int base(){return 11;}}`,
		"src/main/java/app/Derived.java": `package app; import p.*; public class Derived extends Base {public Value field; public Derived(Value value){field=value;} public Value echo(Value input){return input;}}`,
		"src/main/java/app/Main.java":    `package app; import p.*; public class Main {public static void main(String[] args){Value value=new Value(); Derived derived=new Derived(value); System.out.println(derived.field.read()+":"+derived.echo(value).read()+":"+derived.base());}}`,
	}, "app.Main", "7:7:11\n")
}
