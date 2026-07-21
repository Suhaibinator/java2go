package e2e

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// These files deliberately contain a main declaration while remaining invalid
// Java parser/type fixtures. The test below proves that every entry still fails
// javac; if one becomes runnable it must leave this allowlist and gain parity
// coverage automatically.
var intentionallyInvalidMainPrograms = map[string]string{
	"Compass.java":     "duplicate and invalid declarations used by parser tests",
	"LinkedList.java":  "incomplete generic syntax fixture",
	"RandomTypes.java": "collection of intentionally invalid declaration forms",
}

var (
	javaMainDeclaration    = regexp.MustCompile(`(?m)\b(?:public\s+)?static\s+void\s+main\s*\(`)
	javaPackageDeclaration = regexp.MustCompile(`(?m)^\s*package\s+([A-Za-z_$][A-Za-z0-9_$]*(?:\.[A-Za-z_$][A-Za-z0-9_$]*)*)\s*;`)
)

// TestLegacyApplicationParity discovers runnable historical programs that
// predate the e2e and application fixture corpora. This prevents older programs
// with a real main method from silently becoming compile-only translation tests.
func TestLegacyApplicationParity(t *testing.T) {
	repositoryRoot := moduleRoot(t)
	requireApplicationTool(t, "javac")
	requireApplicationTool(t, "java")
	requireApplicationTool(t, "go")

	programs := discoverLegacyApplicationPrograms(t, repositoryRoot)
	if len(programs) == 0 {
		t.Fatal("no historical main-bearing Java programs found outside the designated parity corpora")
	}

	goCache := filepath.Join(t.TempDir(), "go-cache")
	if err := os.MkdirAll(goCache, 0o755); err != nil {
		t.Fatalf("creating shared Go build cache: %v", err)
	}

	// The transpiler symbol table is process-global, so these subtests remain
	// sequential. runApplicationTranspiler resets it around every conversion.
	for _, program := range programs {
		program := program
		t.Run(strings.TrimSuffix(program.relativePath, ".java"), func(t *testing.T) {
			invalidReason, intentionallyInvalid := intentionallyInvalidMainPrograms[program.relativePath]
			if intentionallyInvalid {
				assertLegacyProgramRemainsInvalid(t, program, invalidReason)
				return
			}
			runLegacyApplicationParity(t, repositoryRoot, goCache, program)
		})
	}
}

type legacyApplicationProgram struct {
	path         string
	relativePath string
	packageName  string
	mainClass    string
}

func discoverLegacyApplicationPrograms(t testing.TB, repositoryRoot string) []legacyApplicationProgram {
	t.Helper()

	testfilesRoot := filepath.Join(repositoryRoot, "testfiles")
	applicationFixtures := discoverApplicationFixtures(t, repositoryRoot)
	coveredRoots := make([]string, 0, len(applicationFixtures))
	for _, fixture := range applicationFixtures {
		coveredRoots = append(coveredRoots, filepath.Clean(fixture.sourceRoot))
	}

	var programs []legacyApplicationProgram
	foundInvalid := make(map[string]bool, len(intentionallyInvalidMainPrograms))
	err := filepath.WalkDir(testfilesRoot, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".java" {
			return nil
		}
		if pathWithinDirectory(filePath, filepath.Join(testfilesRoot, "e2e")) {
			return nil
		}
		for _, coveredRoot := range coveredRoots {
			if pathWithinDirectory(filePath, coveredRoot) {
				return nil
			}
		}

		source, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		if !javaMainDeclaration.Match(source) {
			return nil
		}

		relativePath, err := filepath.Rel(testfilesRoot, filePath)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		packageName := ""
		if match := javaPackageDeclaration.FindSubmatch(source); match != nil {
			packageName = string(match[1])
		}
		className := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
		mainClass := className
		if packageName != "" {
			mainClass = packageName + "." + className
		}

		programs = append(programs, legacyApplicationProgram{
			path:         filePath,
			relativePath: relativePath,
			packageName:  packageName,
			mainClass:    mainClass,
		})
		if _, ok := intentionallyInvalidMainPrograms[relativePath]; ok {
			foundInvalid[relativePath] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("discovering historical Java programs: %v", err)
	}

	for invalidPath := range intentionallyInvalidMainPrograms {
		if !foundInvalid[invalidPath] {
			t.Fatalf("invalid-main allowlist entry %q no longer identifies a discovered main-bearing Java file; remove or correct the stale exclusion", invalidPath)
		}
	}
	sort.Slice(programs, func(i, j int) bool { return programs[i].relativePath < programs[j].relativePath })
	return programs
}

func pathWithinDirectory(filePath, directory string) bool {
	relative, err := filepath.Rel(filepath.Clean(directory), filepath.Clean(filePath))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func assertLegacyProgramRemainsInvalid(t testing.TB, program legacyApplicationProgram, reason string) {
	t.Helper()

	classes := filepath.Join(t.TempDir(), "classes")
	if err := os.MkdirAll(classes, 0o755); err != nil {
		t.Fatalf("creating javac output directory for %s: %v", program.relativePath, err)
	}
	result := runApplicationCommand(applicationFixtureTimeout, filepath.Dir(program.path), deterministicApplicationEnv(nil),
		"javac", "--release", applicationJavaRelease, "-encoding", "UTF-8", "-d", classes, program.path)
	requireApplicationCommandStarted(t, "javac", result)
	if result.timedOut {
		t.Fatalf("intentionally invalid Java fixture %s timed out instead of producing a compiler diagnostic", program.relativePath)
	}
	if result.exitCode == 0 {
		t.Fatalf("intentionally invalid Java fixture %s now compiles; remove it from intentionallyInvalidMainPrograms so it gains parity coverage\nreason for prior exclusion: %s",
			program.relativePath, reason)
	}
}

func runLegacyApplicationParity(t testing.TB, repositoryRoot, goCache string, program legacyApplicationProgram) {
	t.Helper()

	programRoot := t.TempDir()
	javaClasses := filepath.Join(programRoot, "java-classes")
	javaWork := filepath.Join(programRoot, "java-work")
	goOutput := filepath.Join(programRoot, "go-output")
	goWork := filepath.Join(programRoot, "go-work")
	for _, directory := range []string{javaClasses, javaWork, goWork} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("creating program directory %s: %v", directory, err)
		}
	}

	javaCompile := runApplicationCommand(applicationFixtureTimeout, filepath.Dir(program.path), deterministicApplicationEnv(nil),
		"javac", "--release", applicationJavaRelease, "-encoding", "UTF-8", "-d", javaClasses, program.path)
	requireApplicationCommandStarted(t, "javac", javaCompile)
	if javaCompile.timedOut || javaCompile.exitCode != 0 {
		t.Fatalf("historical Java program %s did not compile (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
			program.relativePath, javaCompile.exitCode, javaCompile.timedOut, javaCompile.stdout, javaCompile.stderr)
	}

	javaResult := runApplicationCommand(applicationFixtureTimeout, javaWork, deterministicApplicationEnv(nil),
		"java", "-Dfile.encoding=UTF-8", "-Duser.language=en", "-Duser.country=US", "-Duser.timezone=UTC", "-cp", javaClasses, program.mainClass)
	requireApplicationCommandStarted(t, "java", javaResult)
	if javaResult.timedOut || javaResult.exitCode != 0 {
		t.Fatalf("historical Java program %s did not exit successfully (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
			program.relativePath, javaResult.exitCode, javaResult.timedOut, javaResult.stdout, javaResult.stderr)
	}

	modulePath := "parity/legacy"
	if program.packageName != "" {
		modulePath = strings.ReplaceAll(program.packageName, ".", "/")
	}
	fixture := applicationFixture{
		name:       program.relativePath,
		sourceRoot: program.path,
		config: applicationFixtureConfig{
			MainClass:  program.mainClass,
			ModulePath: modulePath,
		},
	}
	if err := runApplicationTranspiler(fixture, goOutput); err != nil {
		t.Fatalf("transpiling historical Java program %s failed: %v", program.relativePath, err)
	}
	if err := configureGeneratedApplicationModule(repositoryRoot, goOutput); err != nil {
		t.Fatalf("configuring generated module for %s: %v", program.relativePath, err)
	}

	buildTarget := "."
	if program.packageName == "" {
		driver := "package main\n\nfunc main() { Main() }\n"
		if err := os.WriteFile(filepath.Join(goOutput, "zz_legacy_driver.go"), []byte(driver), 0o644); err != nil {
			t.Fatalf("writing default-package driver for %s: %v", program.relativePath, err)
		}
	} else {
		if err := writeApplicationDriver(fixture.config, goOutput); err != nil {
			t.Fatalf("writing packaged driver for %s: %v", program.relativePath, err)
		}
		buildTarget = "./paritydriver"
	}

	goEnv := deterministicApplicationEnv(map[string]string{
		"GOCACHE": goCache,
		"GOWORK":  "off",
		"GOFLAGS": "",
	})
	goModuleCompile := runApplicationCommand(applicationFixtureTimeout, goOutput, goEnv, "go", "build", "-mod=mod", "./...")
	requireApplicationCommandStarted(t, "go build ./...", goModuleCompile)
	if goModuleCompile.timedOut || goModuleCompile.exitCode != 0 {
		t.Fatalf("generated Go module for %s did not compile (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
			program.relativePath, goModuleCompile.exitCode, goModuleCompile.timedOut, goModuleCompile.stdout, goModuleCompile.stderr)
	}

	goBinary := filepath.Join(programRoot, "generated-program")
	goCompile := runApplicationCommand(applicationFixtureTimeout, goOutput, goEnv, "go", "build", "-mod=mod", "-o", goBinary, buildTarget)
	requireApplicationCommandStarted(t, "go build", goCompile)
	if goCompile.timedOut || goCompile.exitCode != 0 {
		t.Fatalf("generated Go program for %s did not compile (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
			program.relativePath, goCompile.exitCode, goCompile.timedOut, goCompile.stdout, goCompile.stderr)
	}

	goResult := runApplicationCommand(applicationFixtureTimeout, goWork, deterministicApplicationEnv(nil), goBinary)
	requireApplicationCommandStarted(t, "generated Go program", goResult)
	if goResult.timedOut || goResult.exitCode != javaResult.exitCode {
		t.Fatalf("runtime exit mismatch for %s (Java=%d, Go=%d, Go timeout=%t)\nGo stdout:\n%s\nGo stderr:\n%s",
			program.relativePath, javaResult.exitCode, goResult.exitCode, goResult.timedOut, goResult.stdout, goResult.stderr)
	}
	if !bytes.Equal(goResult.stdout, javaResult.stdout) || !bytes.Equal(goResult.stderr, javaResult.stderr) {
		detail := formatApplicationOutputDifference("java stdout", javaResult.stdout, "go stdout", goResult.stdout)
		if !bytes.Equal(goResult.stderr, javaResult.stderr) {
			detail += "\n" + formatApplicationOutputDifference("java stderr", javaResult.stderr, "go stderr", goResult.stderr)
		}
		t.Fatalf("Java/Go output mismatch for %s\n%s", program.relativePath, detail)
	}
}
