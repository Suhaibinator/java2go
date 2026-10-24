package symbol_test

import (
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func registeredResolutionFiles(t *testing.T, sources map[string]string) map[string]*symbol.FileScope {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	files := make(map[string]*symbol.FileScope)
	for name, source := range sources {
		file := parsing.SourceFile{Name: name, Source: []byte(source)}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		files[name] = file.ParseSymbols()
	}
	for _, file := range files {
		symbol.AddSymbolsToPackage(file)
	}
	return files
}

func registeredResolutionMustNotPanic(t *testing.T) {
	t.Helper()
	if value := recover(); value != nil {
		t.Errorf("registered source lookup panicked: %v", value)
	}
}

func TestRegisteredPackageClassMembership(t *testing.T) {
	files := registeredResolutionFiles(t, map[string]string{
		"foreign/Integer.java": `package foreign;class Base {} public class Integer extends Base {}`,
		"foreign/Outer.java":   `package foreign;public class Outer {public static class Inner {public static class Deep {}}}`,
		"other/Integer.java":   `package other;public class Integer {}`,
	})
	pkg := symbol.GlobalScope.FindPackage("foreign")
	integer := files["foreign/Integer.java"].TopLevelClasses[1]
	outer := files["foreign/Outer.java"].TopLevelClasses[0]
	for _, row := range []struct {
		name string
		want *symbol.ClassScope
	}{
		{"Base", files["foreign/Integer.java"].TopLevelClasses[0]},
		{"Integer", integer}, {"Outer", outer}, {"Inner", outer.Subclasses[0]}, {"Deep", outer.Subclasses[0].Subclasses[0]}, {"Missing", nil},
	} {
		t.Run(row.name, func(t *testing.T) {
			defer registeredResolutionMustNotPanic(t)
			got := pkg.FindClass(row.name)
			t.Logf("name=%q actual=%p expected=%p registeredTopLevels=%d", row.name, got, row.want, len(files["foreign/Integer.java"].TopLevelClasses))
			if got != row.want {
				t.Errorf("lookup must return exact registered declaration scope for %q", row.name)
			}
		})
	}
	t.Run("package_namespace", func(t *testing.T) {
		defer registeredResolutionMustNotPanic(t)
		if got := symbol.GlobalScope.FindPackage("other").FindClass("Integer"); got != files["other/Integer.java"].TopLevelClasses[0] || got == integer {
			t.Error("different source packages were conflated")
		}
	})
	t.Run("nil_package", func(t *testing.T) {
		defer registeredResolutionMustNotPanic(t)
		var absent *symbol.PackageScope
		if absent.FindClass("Integer") != nil {
			t.Error("nil package produced a class")
		}
	})
	t.Run("metadata_and_nil_files", func(t *testing.T) {
		defer registeredResolutionMustNotPanic(t)
		sparse := &symbol.PackageScope{Files: map[string]*symbol.FileScope{"metadata": {}, "nil": nil}}
		if sparse.FindClass("Integer") != nil {
			t.Error("missing source declarations produced class")
		}
	})
}

func TestResolveDefinitionDeclaredSourceMembership(t *testing.T) {
	for _, row := range []struct{ name, donorFile, donor, app, target string }{
		{"secondary_import", "foreign/Integer.java", `package foreign;class Base {} public class Integer extends Base {}`, `package app;import foreign.Integer;public class Use {void accept(Integer value){}}`, "Integer"},
		{"primary_import", "foreign/Integer.java", `package foreign;public class Integer {}`, `package app;import foreign.Integer;public class Use {void accept(Integer value){}}`, "Integer"},
		{"secondary_local", "foreign/Integer.java", `package foreign;public class Integer {}`, `package app;class Base {} public class Use {void accept(Use value){}}`, "Use"},
		{"nested_import", "foreign/Outer.java", `package foreign;public class Outer {public static class Inner {}}`, `package app;import foreign.Outer.Inner;public class Use {void accept(Inner value){}}`, "Inner"},
	} {
		t.Run(row.name, func(t *testing.T) {
			defer registeredResolutionMustNotPanic(t)
			files := registeredResolutionFiles(t, map[string]string{row.donorFile: row.donor, "app/Use.java": row.app})
			file := files["app/Use.java"]
			use := file.FindClassScope("Use")
			method := use.FindMethodByName("accept", nil)
			if method == nil || len(method.Parameters) != 1 {
				t.Fatal("actual declared formal missing")
			}
			formal := method.Parameters[0]
			var target *symbol.ClassScope
			if row.target == "Use" {
				target = use
			} else {
				target = files[row.donorFile].FindClassScope(row.target)
			}
			if target == nil || target.Class == nil || method.DeclarationNode == nil || formal.OriginalType != row.target {
				t.Fatal("actual parsed declaration proof missing")
			}
			want := "*" + target.Class.Name
			if !symbol.ResolveDefinition(formal, file) || formal.Type != want {
				t.Errorf("actual source formal type=%q want declared emitted name %q", formal.Type, want)
			}
			if !symbol.ResolveDefinition(formal, file) || formal.Type != want {
				t.Errorf("repeated resolution changed declared source type=%q want=%q", formal.Type, want)
			}
		})
	}
	for _, row := range []struct{ name, app string }{
		{"class_binder", `package app;import foreign.Integer;public class Use<Integer> {void accept(Integer value){}}`},
		{"method_binder", `package app;import foreign.Integer;public class Use {<Integer> void accept(Integer value){}}`},
	} {
		t.Run(row.name, func(t *testing.T) {
			defer registeredResolutionMustNotPanic(t)
			files := registeredResolutionFiles(t, map[string]string{"foreign/Integer.java": `package foreign;public class Integer {}`, "app/Use.java": row.app})
			file := files["app/Use.java"]
			formal := file.BaseClass.FindMethodByName("accept", nil).Parameters[0]
			binder := formal.DirectTypeParameter
			before := formal.Type
			if binder == nil {
				t.Fatal("actual binder declaration pointer missing")
			}
			if symbol.ResolveDefinition(formal, file) || formal.Type != before || formal.DirectTypeParameter != binder {
				t.Error("source-name import replaced actual lexical type parameter")
			}
		})
	}
	t.Run("unknown_import", func(t *testing.T) {
		defer registeredResolutionMustNotPanic(t)
		files := registeredResolutionFiles(t, map[string]string{"foreign/Integer.java": `package foreign;public class Integer {}`})
		f := &symbol.FileScope{Imports: map[string]string{"Missing": "foreign"}, TopLevelClasses: files["foreign/Integer.java"].TopLevelClasses, BaseClass: files["foreign/Integer.java"].BaseClass}
		definition := &symbol.Definition{Type: "Missing", OriginalType: "Missing"}
		if symbol.ResolveDefinition(definition, f) || definition.Type != "Missing" {
			t.Error("unknown imported member resolved or mutated")
		}
	})
	t.Run("unregistered_package", func(t *testing.T) {
		defer registeredResolutionMustNotPanic(t)
		files := registeredResolutionFiles(t, map[string]string{"app/Use.java": `package app;public class Use {}`})
		f := files["app/Use.java"]
		f.Imports["Missing"] = "unknown"
		definition := &symbol.Definition{Type: "Missing", OriginalType: "Missing"}
		if symbol.ResolveDefinition(definition, f) || definition.Type != "Missing" {
			t.Error("unknown package claimed source resolution")
		}
	})
	t.Run("nil_definition", func(t *testing.T) {
		defer registeredResolutionMustNotPanic(t)
		files := registeredResolutionFiles(t, map[string]string{"app/Use.java": `package app;public class Use {}`})
		if symbol.ResolveDefinition(nil, files["app/Use.java"]) {
			t.Error("nil definition resolved")
		}
	})
	t.Run("nil_file", func(t *testing.T) {
		defer registeredResolutionMustNotPanic(t)
		definition := &symbol.Definition{Type: "Missing", OriginalType: "Missing"}
		if symbol.ResolveDefinition(definition, nil) || definition.Type != "Missing" {
			t.Error("nil source file resolved")
		}
	})
}
