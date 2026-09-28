package transpiler

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestProjectPackageLoweringPreservesBindings(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"j_cycle/j_a/a.go": `package a
import b "cycle/b"
type Box[T any] struct { Value T }
type Holder struct { Box int }
const Key = 3
func Identity[T any](value T) T { return value }
func Result() int { return b.Use() }
`,
		"j_cycle/j_b/b.go": `package b
import alias "cycle/a"
type Box[T any] struct { Value T }
type Child struct { *alias.Box[int] }
func Use() int {
 Box := alias.Holder{Box: 7}
 child := Child{Box: &alias.Box[int]{Value: 5}}
 values := map[int]int{alias.Key: 2}
 return child.Box.Value + Box.Box + alias.Identity[int](values[alias.Key])
}
`,
		"j_cycle/j_app/main.go": `package app
import a "cycle/a"
import b "cycle/b"
func Entry() int { return a.Result() + b.Use() }
`,
	}
	for name, source := range files {
		if err := writeProjectFile(filepath.Join(root, name), []byte(source)); err != nil {
			t.Fatal(err)
		}
	}
	mainImport, entry, err := lowerProjectPackages(root, "lowering.test", "", map[string]string{"cycle/a": "cycle.a", "cycle/b": "cycle.b", "cycle/app": "cycle.app"}, "cycle/app", "Entry")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeProjectFile(filepath.Join(root, "go.mod"), []byte("module lowering.test\n\ngo 1.27.0\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeProjectFile(filepath.Join(root, "main_test.go"), []byte("package main\nimport (\"testing\"; app \""+mainImport+"\")\nfunc TestResult(t *testing.T) { if got := app."+entry+"(); got != 28 { t.Fatalf(\"got %d\",got) } }")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "test", "./...")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("lowered package: %v\n%s", err, output)
	}
}
