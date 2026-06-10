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
	"inheritance/Inheritance":  "package-private class casing: undefined Shape / Newrectangle / Newsquare (superclass embed + super-ctor refs miscased). Array-literal part fixed (task #10 item 5) (ROADMAP §6/#7 casing item 13)",
	"interfaces/Interfaces":    "package-private interface name casing: undefined Greeter. Array-literal part fixed (ROADMAP §6/#7 casing item 13)",
	"lambdas/Lambdas":          "int/int32: 'count' is Go int but return type is int32 (count used as int32 in return). Array-literal part fixed (ROADMAP §6 int typing, item 12)",
	"enums/Enums":              "enum METHOD-name casing: call site emits .ordinal() but generated method is .Ordinal() (WED.ordinal undefined). Enum-constant access fixed (task #10 item 8) (ROADMAP §6/#7 casing item 13)",
	"numeric_edge/NumericEdge": "int overflow not wrapped (int locals are untyped Go consts, not int32); ~0 prints 4294967295; (char) cast prints code point 66 not 'B'; long shift over-masked to 5 bits so 1L<<32 yields 1 (ROADMAP §6, task #10 items 11-12)",

	"var_infer/VarInfer":                   "int/int32 mismatch (untyped local int vs int32 method/return). Array-literal part fixed; var itself works (ROADMAP §6 int typing, item 12)",
	"switch_expr/SwitchExpr":               "switch expressions emit panic() used as a value; loop-var int vs method int32 mismatch (ROADMAP §5)",
	"instanceof_pattern/InstanceofPattern": "instanceof pattern binding not supported; emits invalid composite literal (ROADMAP §5)",
	"records/Records":                      "record constructor call emits undefined ConstructPoint() (ROADMAP §5)",
	"textblocks/TextBlocks":                "text blocks emit a raw multi-line Go string with literal newlines (invalid) (ROADMAP §5)",
	"collections/CollectionOps":            "List/Map intrinsics ARE wired now; blocked by int/int32 typing only (i*i is int but nums.Add wants int32; sum += n mixes int and int32) (ROADMAP §6 int typing)",
	"collections/Optionals":                "Optional.empty() cannot infer T (needs type param at empty() site); Optional.map() unimplemented (num.map_ undefined); Optional.of(10) infers element type any not int (ROADMAP §2/§3)",

	"var_simple/VarSimple":    "var works and String.length() maps on string literals/vars, but a var inferred from a string-CONCAT expression loses its String type so .length() is left unmapped (ROADMAP §2/§6)",
	"concurrency/SyncCounter": "Thread+synchronized now lower (task #11), but blocked by: anonymous Runnable inside a loop (undefined: Runnable, undefined: i captured) + int/int32 (K1) in the run() body (ROADMAP §7 anon-Runnable, §6 int typing)",
	"concurrency/ThreadJoin":  "Thread-subclass->goroutine + sized arrays now work; blocked ENTIRELY by int/int32 (K1): loop vars/fields/args int vs int32 (ROADMAP §6 int typing)",

	"nested/AnonLocal":        "SAM anon class captures local typed int not int32 (value+bump mismatch); method-local class field not emitted (undefined: n) (ROADMAP §4 M4/M5, §6 int typing)",
	"static_init/StaticInit":  "user method named init() collides with Go's reserved init func (emitted as func init with args/return -> invalid); needs name escaping. Also static-init ordering unverified (ROADMAP §4 naming, §6)",
	"overloading/Overloading": "overload resolution collapses all calls to the first overload (describe0/int32); long/double/String/arity dispatch by arg type not implemented (ROADMAP §6/§7)",
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
