package transpiler

import (
	"bytes"
	"fmt"
	"github.com/NickyBoy89/java2go/symbol"
	"os"
	"path/filepath"
	"testing"
)

// Emits a NEW family control alongside the root's frozen original witnesses.
// Execution is a separately authorized generated/race workload; this producer
// never launches Go or JVM itself and never changes the original fixture files.
func TestGenerateAnonymousVolatileWitnessTDD(t *testing.T) {
	withCleanDiagnostics(t)
	root := os.Getenv("JAVA2GO_VOLATILE_GENERATED_DIR")
	if root == "" {
		root = t.TempDir()
	}
	target := filepath.Join(root, "anonymous_final_declaration")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	defer func() { symbol.GlobalScope = previous }()
	var output bytes.Buffer
	if err := runInternal([]string{"-strict", "-sync", "-w", "-output", target, filepath.Join("testdata", "anonymous_volatile_prerequisite", "AnonymousVolatileFieldWitness.java")}, &output, false); err != nil {
		t.Fatalf("actual strict producer: %v; output=%s", err, output.Bytes())
	}
	if len(Diagnostics()) != 0 {
		t.Fatalf("anonymous field witness retained diagnostics: %v", Diagnostics())
	}
	pkg := symbol.GlobalScope.FindPackage("")
	if pkg == nil {
		t.Fatal("missing complete anonymous witness source package")
	}
	entry := ""
	for _, file := range pkg.Files {
		if class := file.FindClassScope("AnonymousVolatileFieldWitness"); class != nil {
			for _, method := range class.Methods {
				if method.OriginalName == "run" && method.IsStatic && len(method.Parameters) == 0 {
					if entry != "" {
						t.Fatal("ambiguous anonymous witness run entry")
					}
					entry = symbol.GoIdentifier(method.Name)
				}
			}
		}
	}
	if entry == "" {
		t.Fatal("original anonymous witness run entry missing")
	}
	// Nine is a source-derived predicate, not an already observed JVM oracle.
	test := fmt.Sprintf("package main\nimport \"testing\"\nfunc TestAnonymousVolatileFieldValues(t *testing.T){if got:=%s();got!=9{t.Fatalf(\"anonymous final field identity/order changed: %%v\",got)}}\n", entry)
	if err := os.WriteFile(filepath.Join(target, "zz_anonymous_volatile_witness_test.go"), []byte(test), 0644); err != nil {
		t.Fatal(err)
	}
}
