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
type NumberMap map[int]int
const Key = 3
func Identity[T any](value T) T { return value }
func Result() int { return b.Use() }
func Keys() int {
 values := map[int]int{Key: 4}
 array := [5]int{Key: 6}
 slice := []int{Key: 8}
 return values[Key] + array[Key] + slice[Key]
}
func Shadow() int {
 b := struct{ Use int }{Use: 9}
 Key := 1
 values := map[int]int{Key: b.Use}
 return values[Key]
}
`,
		"j_cycle/j_b/b.go": `package b
import alias "cycle/a"
type Box[T any] struct { Value T }
type Child struct { *alias.Box[int] }
const Key = 4
type Local struct{}
func Use() int {
 Box := alias.Holder{Box: 7}
 child := Child{Box: &alias.Box[int]{Value: 5}}
 values := map[int]int{alias.Key: 2}
 imported := alias.NumberMap{Key: 0}
 Local := alias.Box[Local]{}
 _ = Local
 _ = imported
 return child.Box.Value + Box.Box + alias.Identity[int](values[alias.Key])
}
`,
		"j_cycle/j_app/main.go": `package app
import a "cycle/a"
import b "cycle/b"
func Entry() int { return a.Result() + b.Use() + a.Keys() + a.Shadow() + a.CrossFile() }
`,
	}
	files["j_cycle/j_a/other.go"] = `package a
func CrossFile() int { return Key }
`
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
	if err := writeProjectFile(filepath.Join(root, "main_test.go"), []byte("package main\nimport (\"testing\"; app \""+mainImport+"\")\nfunc TestResult(t *testing.T) { if got := app."+entry+"(); got != 58 { t.Fatalf(\"got %d\",got) } }")); err != nil {
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

// Imports in a later file must not expose an earlier file's uninitialized
// TypeName to another checker traversing the same Java package cycle.
func TestProjectPackageLoweringForwardImportedType(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"j_cycle/j_a/a.go": `package a
type Value struct { Count int }
`,
		"j_cycle/j_a/z.go": `package a
import b "cycle/b"
func Result() int { return b.Use(Value{Count: 7}) }
`,
		"j_cycle/j_b/b.go": `package b
import a "cycle/a"
func Use(value a.Value) int { return value.Count }
`,
	}
	for name, source := range files {
		if err := writeProjectFile(filepath.Join(root, name), []byte(source)); err != nil {
			t.Fatal(err)
		}
	}
	mainImport, entry, err := lowerProjectPackages(root, "lowering.test", "", map[string]string{"cycle/a": "cycle.a", "cycle/b": "cycle.b"}, "cycle/a", "Result")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeProjectFile(filepath.Join(root, "go.mod"), []byte("module lowering.test\n\ngo 1.27.0\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeProjectFile(filepath.Join(root, "main_test.go"), []byte("package main\nimport (\"testing\"; app \""+mainImport+"\")\nfunc TestResult(t *testing.T) { if got := app."+entry+"(); got != 7 { t.Fatalf(\"got %d\", got) } }")); err != nil {
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
