package transpiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// This test invokes the actual strict standalone producer in-process. It never
// launches Go/JVM commands. Its emitted packages are type-checked and raced by
// the separately sealed third Go workload.
func TestGenerateOperatorFunctionalWitnessesTDD(t *testing.T) {
	root := os.Getenv("JAVA2GO_OPERATOR_FUNCTIONAL_GENERATED_DIR")
	if root == "" {
		root = t.TempDir()
	}
	runtime := os.Getenv("JAVA2GO_OPERATOR_FUNCTIONAL_RUNTIME_DIR")
	if runtime == "" {
		runtime = repoRootDir(t)
	}
	fixtures := []struct{ class, name string }{
		{"OperatorAncestryProbe", "ancestry"},
		{"MultipleSamViewsProbe", "multiple"},
		{"OperatorExecutionCollisionProbe", "collision"},
		{"DefaultParentInferenceProbe", "default_parent"},
	}
	emitted := map[string]string{}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			sourcePath := filepath.Join("testdata", "operator_functional_prerequisite", fixture.class+".java")
			target := filepath.Join(root, fixture.name)
			if err := os.MkdirAll(target, 0755); err != nil {
				t.Fatal(err)
			}
			previous := symbol.GlobalScope
			symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
			defer func() { symbol.GlobalScope = previous }()
			var conversion bytes.Buffer
			if err := runInternal([]string{"-strict", "-sync", "-w", "-output", target, sourcePath}, &conversion, false); err != nil {
				t.Fatalf("strict producer failed: %v; output=%s", err, conversion.Bytes())
			}
			if diagnostics := Diagnostics(); len(diagnostics) != 0 {
				t.Fatalf("strict producer retained diagnostics: %v", diagnostics)
			}
			pkg := symbol.GlobalScope.FindPackage("")
			if pkg == nil {
				t.Fatal("missing complete standalone source package")
			}
			entry := ""
			for _, file := range pkg.Files {
				if class := file.FindClassScope(fixture.class); class != nil {
					for _, method := range class.Methods {
						if projectMain(method) {
							if entry != "" {
								t.Fatal("multiple original public main entries")
							}
							entry = symbol.GoIdentifier(method.Name)
						}
					}
				}
			}
			if entry == "" {
				t.Fatal("original main entry missing")
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(target, fixture.class+".go"), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			entries := 0
			for _, decl := range parsed.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == entry {
					if fn.Recv != nil || fn.Type.Params.NumFields() != 0 {
						t.Fatal("standalone main boundary must be zero argument")
					}
					entries++
				}
			}
			if entries != 1 {
				t.Fatal("original main entry is not unique")
			}
			expected, err := os.ReadFile(filepath.Join("testdata", "operator_functional_prerequisite", fixture.class+".expected.txt"))
			if err != nil {
				t.Fatal(err)
			}
			testSource := fmt.Sprintf(`package main
import("bytes";"io";"os";"testing")
func TestOperatorFunctionalWitness(t *testing.T){
 original:=os.Stdout
 reader,writer,err:=os.Pipe();if err!=nil{t.Fatal(err)}
 defer reader.Close()
 os.Stdout=writer
 defer func(){os.Stdout=original;writer.Close()}()
 received:=make(chan []byte,1)
 go func(){var data bytes.Buffer;if _,err:=io.Copy(&data,reader);err!=nil{received<-nil;return};received<-data.Bytes()}()
 %s()
 if err:=writer.Close();err!=nil{t.Fatal(err)}
 os.Stdout=original
 got:=<-received
 want:=[]byte(%q)
 if !bytes.Equal(got,want){t.Fatalf("generated operator_functional stdout %%q != frozen expectation %%q",got,want)}
}
`, entry, string(expected))
			if err := os.WriteFile(filepath.Join(target, "zz_operator_functional_witness_test.go"), []byte(testSource), 0644); err != nil {
				t.Fatal(err)
			}
			emitted[fixture.name] = entry
		})
	}
	if len(emitted) != len(fixtures) {
		t.Fatal("generation omitted a frozen witness")
	}
	module := fmt.Sprintf("module java2go.operator_functional.generated\n\ngo 1.27.0\n\nrequire (\n github.com/NickyBoy89/java2go v0.0.0\n golang.org/x/exp v0.0.0-20260824195058-e88cd73687aa\n golang.org/x/text v0.42.0\n)\nreplace github.com/NickyBoy89/java2go => %s\n", strconv.Quote(filepath.ToSlash(runtime)))
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(module), 0644); err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile(filepath.Join(runtime, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.sum"), sum, 0644); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(emitted, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "entry-inventory.json"), append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
