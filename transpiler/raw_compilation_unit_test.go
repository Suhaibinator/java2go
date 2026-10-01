package transpiler

import (
	"bytes"
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func rawCompilationUnitRun(args []string) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("unexpected registration panic: %v", value)
		}
	}()
	return run(args, &bytes.Buffer{})
}

func rawCompilationUnitGlobals(t *testing.T) {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous; setStrictMode(false); resetDiagnostics() })
}

func TestRawCompilationUnitPackageMetadataRetained(t *testing.T) {
	for _, concurrent := range []bool{false, true} {
		t.Run(fmt.Sprintf("concurrent=%t", concurrent), func(t *testing.T) {
			rawCompilationUnitGlobals(t)
			input := filepath.Join("testdata", "raw_compilation_units", "positive")
			output := filepath.Join(t.TempDir(), "generated")
			args := []string{"-strict", "-w", "-output", output, "-module", "raw"}
			if !concurrent {
				args = append(args, "-sync")
			}
			args = append(args, input)
			if err := rawCompilationUnitRun(args); err != nil {
				t.Fatalf("all package metadata and class sources must ingest: %v", err)
			}
			wanted := map[string]string{
				"alpha/package-info.go": "alpha", "alpha/Value.go": "alpha",
				"beta/package-info.go": "beta", "beta/Consumer.go": "beta",
				"gamma/Metadata.go": "gamma", "gamma/Gamma.go": "gamma", "meta/Scope.go": "meta",
			}
			for relative, pkg := range wanted {
				path := filepath.Join(output, filepath.FromSlash(relative))
				unit, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
				if err != nil {
					t.Fatalf("every input must emit valid Go: %s: %v", relative, err)
				}
				if unit.Name.Name != pkg {
					t.Errorf("package %s = %q; want %q", relative, unit.Name.Name, pkg)
				}
				if strings.Contains(relative, "package-info") || strings.HasSuffix(relative, "Metadata.go") {
					if len(unit.Decls) != 0 {
						t.Errorf("package metadata generated executable declarations: %s", relative)
					}
					content, _ := os.ReadFile(path)
					if !bytes.Contains(content, []byte("Java package metadata")) {
						t.Errorf("metadata acknowledgement missing: %s", relative)
					}
				}
			}
			for _, item := range []struct{ pkg, class string }{{"raw.alpha", "Value"}, {"raw.alpha", "Helper"}, {"raw.beta", "Consumer"}, {"raw.gamma", "Gamma"}, {"raw.meta", "Scope"}} {
				pkg := symbol.GlobalScope.FindPackage(item.pkg)
				if pkg == nil || pkg.FindClassScope(item.class) == nil {
					t.Errorf("adjacent source class missing: %s.%s", item.pkg, item.class)
				}
			}
			for relative, annotation := range map[string]string{"alpha/package-info.go": "@Scope(\"domain\")", "beta/package-info.go": "@raw.meta.Scope(\"consumer\")"} {
				content, err := os.ReadFile(filepath.Join(output, relative))
				if err != nil || !bytes.Contains(content, []byte(annotation)) {
					t.Errorf("original package annotation not retained: %s: %v", relative, err)
				}
			}
			count := 0
			if err := filepath.WalkDir(output, func(path string, entry os.DirEntry, err error) error {
				if err == nil && !entry.IsDir() && filepath.Ext(path) == ".go" {
					count++
				}
				return err
			}); err != nil {
				t.Fatal(err)
			}
			if count != len(wanted) {
				t.Fatalf("input-to-output inventory count = %d; want %d", count, len(wanted))
			}
		})
	}
}

func TestRawCompilationUnitNamesDoNotClassifyContent(t *testing.T) {
	rawCompilationUnitGlobals(t)
	input := filepath.Join("testdata", "raw_compilation_units", "class_named_metadata")
	output := filepath.Join(t.TempDir(), "generated")
	if err := rawCompilationUnitRun([]string{"-strict", "-sync", "-w", "-module", "raw", "-output", output, input}); err != nil {
		t.Fatal(err)
	}
	unit, err := parser.ParseFile(token.NewFileSet(), filepath.Join(output, "names", "package-info.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	goast.Inspect(unit, func(node goast.Node) bool {
		if _, ok := node.(*goast.TypeSpec); ok {
			found = true
		}
		return true
	})
	if !found {
		t.Fatal("class-bearing package-info source was silently dropped")
	}
}

func TestRawCompilationUnitExplicitDiagnosticsBeforeWrites(t *testing.T) {
	for _, test := range []struct{ directory, message string }{
		{"module", "module_declaration"}, {"module_other_name", "module_declaration"},
		{"empty", "empty Java compilation unit"}, {"invalid", "Java parse error"},
		{"import_only", "no type or package declaration"},
	} {
		t.Run(test.directory, func(t *testing.T) {
			rawCompilationUnitGlobals(t)
			input := filepath.Join("testdata", "raw_compilation_units", test.directory)
			output := filepath.Join(t.TempDir(), "not-created")
			err := rawCompilationUnitRun([]string{"-strict", "-sync", "-w", "-output", output, input})
			if err == nil || !strings.Contains(err.Error(), test.message) || strings.Contains(err.Error(), "panic") {
				t.Fatalf("want explicit %q diagnostic; got %v", test.message, err)
			}
			if !strings.Contains(err.Error(), ".java") {
				t.Errorf("diagnostic omitted source filename: %v", err)
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Errorf("invalid unit wrote output: %v", err)
			}
			if len(symbol.GlobalScope.Packages) != 0 {
				t.Errorf("invalid unit registered symbols: %#v", symbol.GlobalScope.Packages)
			}
		})
	}
}
