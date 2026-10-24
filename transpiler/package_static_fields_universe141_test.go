package transpiler

import (
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func packageFieldsUniverse141Files(t *testing.T) map[string]string {
	t.Helper()
	root := filepath.Join("testdata", "package_fields_universe141", "project")
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestPackageStaticFieldsUniverse141CompleteSeededJDK21(t *testing.T) {
	files := packageFieldsUniverse141Files(t)
	for _, seed := range []string{"17", "41", "97"} {
		t.Run("seed"+seed, func(t *testing.T) {
			expected, err := os.ReadFile(filepath.Join("testdata", "package_fields_universe141", "oracles", "seed-"+seed+".stdout"))
			if err != nil {
				t.Fatal(err)
			}
			runCampaignCompilerStrictProjectOracle47Args(t, files, "org.packagefields.app.Main", string(expected), seed)
		})
	}
}

func TestPackageStaticFieldsUniverse141DeclarationAndOrder(t *testing.T) {
	sources := packageFieldsUniverse141Files(t)
	var paths []string
	for path := range sources {
		if strings.HasSuffix(path, ".java") {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	check := func(paths []string) map[string]string {
		previous := symbol.GlobalScope
		symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
		defer func() { symbol.GlobalScope = previous }()
		var files []parsing.SourceFile
		type facts struct {
			originalName, originalType string
			declaration                any
			parameter                  *symbol.TypeParamDeclaration
			name                       string
		}
		original := map[*symbol.Definition]facts{}
		for _, path := range paths {
			file := parsing.SourceFile{Name: path, Source: []byte(sources[path])}
			if err := file.ParseAST(); err != nil {
				t.Fatal(err)
			}
			symbols := file.ParseSymbols()
			symbol.AddSymbolsToPackage(symbols)
			for _, owner := range symbols.TopLevelClasses {
				for _, field := range owner.Fields {
					original[field] = facts{field.OriginalName, field.OriginalType, field.DeclarationNode, field.DirectTypeParameter, field.Name}
				}
			}
			files = append(files, file)
		}
		ResolveFiles(files)
		allocated := map[string]string{}
		occupied := map[string]map[string]bool{}
		for _, file := range files {
			if occupied[file.Symbols.Package] == nil {
				occupied[file.Symbols.Package] = map[string]bool{}
			}
			for _, owner := range file.Symbols.TopLevelClasses {
				for _, field := range owner.Fields {
					before := original[field]
					if field.OriginalName != before.originalName || field.OriginalType != before.originalType || field.DeclarationNode != before.declaration || field.DirectTypeParameter != before.parameter {
						t.Fatal("allocation changed Java declaration facts")
					}
					if !field.IsStatic {
						if field.Name != before.name {
							t.Fatalf("safe receiver selector %s renamed to %s", before.name, field.Name)
						}
						continue
					}
					emitted := symbol.GoIdentifier(field.Name)
					if types.Universe.Lookup(emitted) != nil {
						t.Errorf("package field %s.%s captured Go universe binding %s", owner.Class.OriginalName, field.OriginalName, emitted)
					}
					if occupied[file.Symbols.Package][emitted] {
						t.Errorf("duplicate package field %s", emitted)
					}
					occupied[file.Symbols.Package][emitted] = true
					allocated[file.Symbols.Package+"."+owner.Class.OriginalName+"."+field.OriginalName] = emitted
				}
			}
		}
		ResolveFiles(files)
		for _, file := range files {
			for _, owner := range file.Symbols.TopLevelClasses {
				for _, field := range owner.Fields {
					if field.IsStatic {
						key := file.Symbols.Package + "." + owner.Class.OriginalName + "." + field.OriginalName
						if allocated[key] != symbol.GoIdentifier(field.Name) {
							t.Fatalf("second resolution changed %s", key)
						}
					}
				}
			}
		}
		return allocated
	}
	forward := check(paths)
	reverse := append([]string(nil), paths...)
	for i, j := 0, len(reverse)-1; i < j; i, j = i+1, j-1 {
		reverse[i], reverse[j] = reverse[j], reverse[i]
	}
	if !reflect.DeepEqual(forward, check(reverse)) {
		t.Fatalf("file order changed allocation: %v", forward)
	}
}
