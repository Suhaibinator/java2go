// Package e2e contains end-to-end tests that take a self-contained Java program,
// compile and run it with the real JDK, transpile it with java2go, compile and run
// the generated Go, and require byte-for-byte parity of exit code, stdout, and
// stderr. The checked-in .expected file is also verified against the live Java
// oracle so a stale snapshot cannot mask a Java/Go mismatch.
//
// Each program lives in testfiles/e2e/<feature>/<Name>.java with a sibling
// <Name>.expected. Every discovered program is mandatory; the suite has no skip
// list or output normalization.
package e2e

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	// The transpiler logs progress through the global logrus logger; silence it so
	// only test results show.
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func moduleRoot(t testing.TB) string {
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
	requireApplicationTool(t, "javac")
	requireApplicationTool(t, "java")
	requireApplicationTool(t, "go")

	buildRoot := t.TempDir()
	goCache := filepath.Join(buildRoot, "go-cache")
	if err := os.MkdirAll(goCache, 0o755); err != nil {
		t.Fatalf("creating shared Go build cache: %v", err)
	}

	keys := make([]string, 0, len(programs))
	for key := range programs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// The transpiler keeps a process-global symbol table. Keep these subtests
	// sequential; runApplicationTranspiler resets that table for every program.
	for _, key := range keys {
		javaPath := programs[key]
		t.Run(key, func(t *testing.T) {
			runProgram(t, root, goCache, key, javaPath)
		})
	}
}

func runProgram(t *testing.T, root, goCache, key, javaPath string) {
	t.Helper()

	expectedPath := strings.TrimSuffix(javaPath, ".java") + ".expected"
	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("reading expected output %s: %v", expectedPath, err)
	}

	programRoot := t.TempDir()
	javaClasses := filepath.Join(programRoot, "java-classes")
	javaWork := filepath.Join(programRoot, "java-work")
	goOutput := filepath.Join(programRoot, "go-output")
	goWork := filepath.Join(programRoot, "go-work")
	for _, directory := range []string{javaClasses, javaWork, goWork} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("creating program build directory %s: %v", directory, err)
		}
	}

	javaCompile := runApplicationCommand(applicationFixtureTimeout, filepath.Dir(javaPath), deterministicApplicationEnv(nil),
		"javac", "--release", applicationJavaRelease, "-encoding", "UTF-8", "-d", javaClasses, javaPath)
	requireApplicationCommandStarted(t, "javac", javaCompile)
	if javaCompile.timedOut || javaCompile.exitCode != 0 {
		t.Fatalf("Java oracle for %s did not compile (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
			key, javaCompile.exitCode, javaCompile.timedOut, javaCompile.stdout, javaCompile.stderr)
	}

	mainClass := strings.TrimSuffix(filepath.Base(javaPath), filepath.Ext(javaPath))
	javaResult := runApplicationCommand(applicationFixtureTimeout, javaWork, deterministicApplicationEnv(nil),
		"java", "-Dfile.encoding=UTF-8", "-Duser.language=en", "-Duser.country=US", "-Duser.timezone=UTC", "-cp", javaClasses, mainClass)
	requireApplicationCommandStarted(t, "java", javaResult)
	if javaResult.timedOut || javaResult.exitCode != 0 {
		t.Fatalf("Java oracle for %s did not exit successfully (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
			key, javaResult.exitCode, javaResult.timedOut, javaResult.stdout, javaResult.stderr)
	}
	if !bytes.Equal(javaResult.stdout, expectedBytes) {
		t.Fatalf("checked-in oracle for %s does not exactly match live Java output\n%s",
			key, formatApplicationOutputDifference("expected", expectedBytes, "java stdout", javaResult.stdout))
	}
	if len(javaResult.stderr) != 0 {
		t.Fatalf("Java oracle for %s wrote unexpected stderr\n%s",
			key, formatApplicationOutputDifference("expected stderr", nil, "java stderr", javaResult.stderr))
	}

	fixture := applicationFixture{
		name:       key,
		sourceRoot: javaPath,
		config: applicationFixtureConfig{
			ModulePath: "parity/e2e",
		},
	}
	if err := runApplicationTranspiler(fixture, goOutput); err != nil {
		t.Fatalf("transpiling %s failed: %v", javaPath, err)
	}
	if err := configureGeneratedApplicationModule(root, goOutput); err != nil {
		t.Fatalf("configuring generated module for %s: %v", key, err)
	}

	// The transpiler renames Java's main() to an exported Main(); add a driver so
	// the generated package is runnable.
	driver := "package main\n\nfunc main() { Main() }\n"
	if err := os.WriteFile(filepath.Join(goOutput, "zz_e2e_driver.go"), []byte(driver), 0o644); err != nil {
		t.Fatalf("writing driver: %v", err)
	}

	goEnv := deterministicApplicationEnv(map[string]string{
		"GOCACHE": goCache,
		"GOWORK":  "off",
		"GOFLAGS": "",
	})
	goBinary := filepath.Join(programRoot, "generated-program")
	goCompile := runApplicationCommand(applicationFixtureTimeout, goOutput, goEnv, "go", "build", "-mod=mod", "-o", goBinary, ".")
	requireApplicationCommandStarted(t, "go build", goCompile)
	if goCompile.timedOut || goCompile.exitCode != 0 {
		t.Fatalf("generated Go for %s did not compile (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
			key, goCompile.exitCode, goCompile.timedOut, goCompile.stdout, goCompile.stderr)
	}

	goResult := runApplicationCommand(applicationFixtureTimeout, goWork, deterministicApplicationEnv(nil), goBinary)
	requireApplicationCommandStarted(t, "generated Go program", goResult)
	if goResult.timedOut || goResult.exitCode != javaResult.exitCode {
		t.Fatalf("runtime exit mismatch for %s (Java=%d, Go=%d, Go timeout=%t)\nGo stdout:\n%s\nGo stderr:\n%s",
			key, javaResult.exitCode, goResult.exitCode, goResult.timedOut, goResult.stdout, goResult.stderr)
	}
	if !bytes.Equal(goResult.stdout, javaResult.stdout) || !bytes.Equal(goResult.stderr, javaResult.stderr) {
		detail := formatApplicationOutputDifference("java stdout", javaResult.stdout, "go stdout", goResult.stdout)
		if !bytes.Equal(goResult.stderr, javaResult.stderr) {
			detail += "\n" + formatApplicationOutputDifference("java stderr", javaResult.stderr, "go stderr", goResult.stderr)
		}
		t.Fatalf("Java/Go output mismatch for %s\n%s", key, detail)
	}
}
