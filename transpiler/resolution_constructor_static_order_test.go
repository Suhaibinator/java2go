package transpiler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

// Static allocation must observe constructor helpers after overload resolution.
// Every input is ordinary Java source; the JVM oracle also checks constructor
// overload selection, the static value and most-derived virtual dispatch.
func TestResolutionBatch_ConstructorHelperStaticFieldOrdering(t *testing.T) {
	fixture := filepath.Join("testdata", "resolution_constructor_static_collision")
	t.Run("batch-names", func(t *testing.T) {
		previous := symbol.GlobalScope
		symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
		t.Cleanup(func() { symbol.GlobalScope = previous })
		files, err := parsing.ReadSourcesInDir(filepath.Join(fixture, "src", "main", "java"))
		if err != nil {
			t.Fatal(err)
		}
		for index := range files {
			if err := files[index].ParseAST(); err != nil {
				t.Fatal(err)
			}
			symbol.AddSymbolsToPackage(files[index].ParseSymbols())
		}
		ResolveFiles(files)
		var constructor, field *symbol.Definition
		for _, file := range files {
			for _, owner := range file.Symbols.TopLevelClasses {
				if owner.Class.OriginalName == "A" {
					for _, method := range owner.Methods {
						if method.Constructor && len(method.Parameters) == 0 {
							constructor = method
						}
					}
				}
				if owner.Class.OriginalName == "B" {
					for _, candidate := range owner.Fields {
						if candidate.OriginalName == "NewA0Java2goWithSelf" {
							field = candidate
						}
					}
				}
			}
		}
		if constructor == nil || field == nil {
			t.Fatal("fixture lost declarations")
		}
		if constructor.Name != "NewA0" {
			t.Fatalf("constructor overload precondition: got %s", constructor.Name)
		}
		if field.Name == constructorWithSelfName(constructor.Name) {
			t.Fatalf("POST_ORDINARY_STATIC_ALLOCATION_MISSING: static field %s collides with final overloaded constructor helper", field.Name)
		}
	})
	t.Run("java-oracle", func(t *testing.T) {
		files := make(map[string]string)
		for _, relative := range []string{"pom.xml", "src/main/java/p/A.java", "src/main/java/p/B.java", "src/main/java/p/Main.java"} {
			source, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(relative)))
			if err != nil {
				t.Fatal(err)
			}
			files[relative] = string(source)
		}
		runCampaignCompilerProjectOracle(t, files, "p.Main", "1:1:3:2\n")
	})
}
