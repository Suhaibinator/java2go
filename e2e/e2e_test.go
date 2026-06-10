// Package e2e contains end-to-end tests that take a self-contained Java program,
// transpile it with java2go, compile the generated Go inside this module (so the
// stdjava runtime import resolves), run it, and compare stdout against a recorded
// .expected file produced by a real `java` binary.
//
// Each program lives in testfiles/e2e/<feature>/<Name>.java with a sibling
// <Name>.expected. Programs that currently fail because of a known transpiler gap
// are listed in skipReasons with a ROADMAP reference; they stay in the suite as
// skipped tests and become regression tests the moment the gap is closed.
package e2e

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	java2go "github.com/NickyBoy89/java2go"
	log "github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	// The transpiler logs progress through the global logrus logger; silence it so
	// only test results show.
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// skipReasons maps a fixture key ("<feature>/<Name>") to the reason it is skipped.
// The reason references the ROADMAP section that, once implemented, should let the
// test pass. Remove an entry here to turn its program into an enforced regression
// test. Keep this list in sync as features land.
var skipReasons = map[string]string{
	"controlflow/ControlFlow":  "switch statements emit UNSUPPORTED panic; ternary and do-while condition emit undefined ternary()/ILLEGAL() helpers (ROADMAP §1 robustness, §5 switch)",
	"arithmetic/Arithmetic":    "postfix/prefix increment used as an expression emits undefined PostUpdate()/PreUpdate() (ROADMAP §1/§6)",
	"inheritance/Inheritance":  "array creation with initializer emits a bare {..} composite literal missing its type; package-private superclass casing mismatch (ROADMAP §6)",
	"interfaces/Interfaces":    "array creation with initializer emits a bare {..} composite literal missing its type (ROADMAP §6)",
	"lambdas/Lambdas":          "array creation with initializer (new int[]{..}) emits a bare {..} composite literal missing its type (ROADMAP §6)",
	"generics/Generics":        "boxed Integer type parameter emits undefined *Integer; autoboxing not mapped (ROADMAP §6)",
	"enums/Enums":              "qualified enum constant access (Day.WED) left unresolved; package-private enum casing mismatch (ROADMAP §6)",
	"exceptions/Exceptions":    "native Go panics (divide-by-zero) not normalized to Java ArithmeticException, so catch does not fire (ROADMAP §3)",
	"strings/Strings":          "String instance methods (trim/split) and StringBuilder construction not mapped to runtime (ROADMAP §2)",
	"numeric_edge/NumericEdge": ">>> emits undefined UnsignedRightShift(); int shift distance not masked to 5 bits (ROADMAP §6)",
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// This test file lives in <root>/e2e, so the module root is one level up.
	return filepath.Dir(wd)
}

func findPrograms(t *testing.T, root string) map[string]string {
	t.Helper()
	base := filepath.Join(root, "testfiles", "e2e")
	programs := make(map[string]string)
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		key := strings.TrimSuffix(filepath.ToSlash(rel), ".java")
		programs[key] = path
		return nil
	})
	if err != nil {
		t.Fatalf("scanning fixtures: %v", err)
	}
	return programs
}

func TestE2EPrograms(t *testing.T) {
	root := moduleRoot(t)
	programs := findPrograms(t, root)
	if len(programs) == 0 {
		t.Fatal("no e2e Java programs found under testfiles/e2e")
	}

	// All generated Go lands under this dot-prefixed dir inside the module, so the
	// stdjava import resolves and `go test ./...` ignores it. Cleared per run.
	buildRoot := filepath.Join(root, "e2e", ".e2ebuild")
	if err := os.RemoveAll(buildRoot); err != nil {
		t.Fatalf("clearing build root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(buildRoot) })

	for key, javaPath := range programs {
		t.Run(key, func(t *testing.T) {
			if reason, skip := skipReasons[key]; skip {
				t.Skip(reason)
			}
			runProgram(t, root, buildRoot, key, javaPath)
		})
	}
}

func runProgram(t *testing.T, root, buildRoot, key, javaPath string) {
	t.Helper()

	expectedPath := strings.TrimSuffix(javaPath, ".java") + ".expected"
	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("reading expected output %s: %v", expectedPath, err)
	}
	expected := string(expectedBytes)

	outDir := filepath.Join(buildRoot, strings.ReplaceAll(key, "/", "_"))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatalf("creating build dir: %v", err)
	}

	// Transpile through the public library API, exactly as the CLI would.
	if err := java2go.Run([]string{"-w", "-output", outDir, javaPath}); err != nil {
		t.Fatalf("transpiling %s failed: %v", javaPath, err)
	}

	// The transpiler renames Java's main() to an exported Main(); add a driver so
	// the generated package is runnable.
	driver := "package main\n\nfunc main() { Main() }\n"
	if err := os.WriteFile(filepath.Join(outDir, "zz_e2e_driver.go"), []byte(driver), 0o644); err != nil {
		t.Fatalf("writing driver: %v", err)
	}

	pkgPath := "./" + filepath.ToSlash(mustRel(t, root, outDir)) + "/"
	cmd := exec.Command("go", "run", pkgPath)
	cmd.Dir = root
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		t.Fatalf("compiling/running generated Go for %s failed: %v\n%s", key, runErr, string(out))
	}

	got := string(out)
	if normalizeOutput(got) != normalizeOutput(expected) {
		t.Fatalf("output mismatch for %s\n--- expected ---\n%s\n--- got ---\n%s", key, expected, got)
	}
}

func mustRel(t *testing.T, base, target string) string {
	t.Helper()
	rel, err := filepath.Rel(base, target)
	if err != nil {
		t.Fatalf("rel path: %v", err)
	}
	return rel
}

// normalizeOutput trims a single trailing newline so a fixture's .expected file
// (which always ends in a newline) compares equal to program stdout.
func normalizeOutput(s string) string {
	return strings.TrimRight(s, "\n")
}
