package transpiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// assertGeneratedCompiles converts Java source and type-checks the Go that comes
// out against the real stdjava runtime.
//
// The intrinsic integration tests otherwise assert only that certain substrings
// appear in the printed Go. That cannot catch an intrinsic which emits a
// well-shaped call the Go compiler then rejects — an unresolved type parameter,
// a lambda with no result type, a method name left in its Java spelling — which
// is exactly how a whole family of registrations was able to look tested while
// generating code that never built. Use this for any intrinsic whose generated
// form depends on inference rather than on a fixed rewrite.
func assertGeneratedCompiles(t *testing.T, javaSource string) string {
	t.Helper()

	generated := renderGoFileFromJava(t, javaSource)
	repoRoot := repoRootDir(t)
	tempDir := t.TempDir()

	goMod := "module generated\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => " + repoRoot + "\n"
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("writing temp go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "generated.go"), []byte(generated), 0o644); err != nil {
		t.Fatalf("writing generated source: %v", err)
	}
	// The transpiler emits `package main`, and a Java `main` becomes an exported
	// Main, so the package has no Go entry point to link. Supply an empty one,
	// as the e2e harness does with its driver.
	entryPoint := "package main" + "\n\n" + "func main() {}" + "\n"
	if err := os.WriteFile(filepath.Join(tempDir, "zz_main.go"), []byte(entryPoint), 0o644); err != nil {
		t.Fatalf("writing entry point: %v", err)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = tempDir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed:\n%s", string(out))
	}

	build := exec.Command("go", "build", "./...")
	build.Dir = tempDir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("generated Go does not compile:\n%s\n--- generated ---\n%s", string(out), generated)
	}
	return generated
}
