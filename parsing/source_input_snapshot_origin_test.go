package parsing

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestSourceInputSnapshotRetainedSourceDomain(t *testing.T) {
	for _, source := range []string{"class \\uu0043anonical {int value;}", "class Repair {Object value;void update(){(value)=null;}}", "class Physical {int value;}\r\n", "package owned;", ""} {
		t.Run(source, func(t *testing.T) {
			raw := []byte(source)
			file := SourceFile{Name: "Owned.java", Source: bytes.Clone(raw)}
			if err := file.ParseAST(); err != nil {
				t.Fatal(err)
			}
			retained := bytes.Clone(file.Source)
			root := file.Ast
			scope := file.ParseSymbols()
			snapshot := scope.SourceInputSnapshot()
			if snapshot == nil || snapshot.Validate() != nil {
				t.Fatal("actual retained source did not validate")
			}
			if scope.ParsedRoot() != root || !bytes.Equal(scope.Source, retained) {
				t.Fatal("attachment changed retained source/root")
			}
			if file.UnicodeSource != nil {
				if !bytes.Equal(scope.OriginalSource, raw) || len(scope.SourceOffsets) != len(retained)+1 {
					t.Fatal("lost original Unicode offset domain")
				}
			}
			file.InputKind = symbol.SourceInputKind(255)
			if err := snapshot.Validate(); err != nil {
				t.Fatal("mutating parser wrapper retargeted existing attachment", err)
			}
		})
	}
}

func TestSourceInputSnapshotPhysicalIngress(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Owned.java")
	source := []byte("class Owned {}")
	if err := os.WriteFile(path, source, 0600); err != nil {
		t.Fatal(err)
	}
	files, err := ReadSourcesInDir(dir)
	if err != nil || len(files) != 1 {
		t.Fatalf("physical source ingress: %v, files=%d", err, len(files))
	}
	file := &files[0]
	if file.InputKind != symbol.PhysicalSourceInput || file.Name != path || !bytes.Equal(file.Source, source) {
		t.Fatal("physical origin/source changed")
	}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	scope := file.ParseSymbols()
	if scope.SourceFilename != path || scope.SourceInputSnapshot() == nil || scope.SourceInputSnapshot().Validate() != nil {
		t.Fatal("physical attachment invalid")
	}
}

func TestSourceInputSnapshotAnonymousNamePreservesParse(t *testing.T) {
	file := SourceFile{Source: []byte("class Anonymous {}")}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	scope := file.ParseSymbols()
	if scope.FindClassScope("Anonymous") == nil || scope.SourceInputSnapshot() == nil || scope.SourceInputSnapshot().Validate() == nil {
		t.Fatal("empty name must preserve parsing while snapshot remains invalid")
	}
}

func TestSourceInputSnapshotActualParseLifetime(t *testing.T) {
	for _, source := range []string{"class Owned {}", "package owned;", `class \uu0043anonical {int value;}`} {
		t.Run(source, func(t *testing.T) {
			owner := func() *symbol.FileScope {
				file := SourceFile{Name: "Lifetime.java", Source: []byte(source)}
				if err := file.ParseAST(); err != nil {
					t.Fatal(err)
				}
				owner := file.ParseSymbols()
				if owner.ParsedRoot() != file.Ast {
					t.Fatal("actual parse root identity changed")
				}
				return owner
			}()
			snapshot := owner.SourceInputSnapshot()
			runtime.GC()
			runtime.GC()
			if owner.ParsedRoot() == nil || owner.ParsedRoot().Type() != "program" || snapshot == nil || snapshot.Validate() != nil {
				t.Fatal("snapshot/owner did not retain actual parsed input lifetime")
			}
		})
	}
}
