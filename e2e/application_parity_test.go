package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	java2go "github.com/NickyBoy89/java2go"
	"github.com/NickyBoy89/java2go/symbol"
)

// Application fixtures exercise project-wide conversion rather than isolated
// language features. Each directory under testfiles/applications must contain:
//
//   - fixture.json: entry point, module path, and parity status
//   - expected.stdout: the exact, deterministic output of the Java program
//   - src/**/*.java: the complete Java source tree, unless fixture.json points
//     source_root at an existing source tree under testfiles
//
// A known-gap fixture is still compiled and run as far as possible. Its observed
// failure must match both the declared stage and identifying text. Once it passes,
// this test fails and asks that the fixture be promoted to "passing".

const (
	applicationFixtureTimeout = 60 * time.Second
	applicationJavaRelease    = "21"
	java2goModulePath         = "github.com/NickyBoy89/java2go"
)

type applicationFixtureConfig struct {
	MainClass               string `json:"main_class"`
	ModulePath              string `json:"module_path"`
	SourceRoot              string `json:"source_root,omitempty"`
	Status                  string `json:"status"`
	KnownGap                string `json:"known_gap,omitempty"`
	ExpectedFailureStage    string `json:"expected_failure_stage,omitempty"`
	ExpectedFailureContains string `json:"expected_failure_contains,omitempty"`
}

type applicationFixture struct {
	name           string
	root           string
	sourceRoot     string
	expectedStdout []byte
	expectedStderr []byte
	config         applicationFixtureConfig
}

type applicationCommandResult struct {
	stdout   []byte
	stderr   []byte
	exitCode int
	err      error
	timedOut bool
}

type applicationParityFailure struct {
	stage  string
	detail string
}

var javaIdentifier = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

func TestApplicationParity(t *testing.T) {
	repositoryRoot := moduleRoot(t)
	requireApplicationTool(t, "javac")
	requireApplicationTool(t, "java")
	requireApplicationTool(t, "go")

	fixtures := discoverApplicationFixtures(t, repositoryRoot)
	strictKnownGaps := os.Getenv("JAVA2GO_PARITY_STRICT") == "1"

	// The transpiler keeps a process-global symbol table, so these subtests must
	// remain sequential. runApplicationTranspiler also resets the table per fixture.
	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			failure := runApplicationFixture(t, repositoryRoot, fixture)

			if fixture.config.Status == "passing" {
				if failure != nil {
					t.Fatalf("application parity failed at %s:\n%s", failure.stage, failure.detail)
				}
				return
			}

			if failure == nil {
				t.Fatalf("known gap unexpectedly passed; promote fixture status to %q and remove the known-gap fields", "passing")
			}
			if failure.stage != fixture.config.ExpectedFailureStage {
				t.Fatalf("known gap changed failure stage: declared %q, observed %q\nknown gap: %s\n%s",
					fixture.config.ExpectedFailureStage, failure.stage, fixture.config.KnownGap, failure.detail)
			}
			if !strings.Contains(failure.detail, fixture.config.ExpectedFailureContains) {
				t.Fatalf("known gap failed at the expected stage but not for the expected reason\nexpected detail to contain: %q\nknown gap: %s\nobserved:\n%s",
					fixture.config.ExpectedFailureContains, fixture.config.KnownGap, failure.detail)
			}
			if strictKnownGaps {
				t.Fatalf("known gap remains and JAVA2GO_PARITY_STRICT=1:\n%s", failure.detail)
			}

			t.Logf("known gap reproduced at %s: %s", failure.stage, fixture.config.KnownGap)
		})
	}
}

func discoverApplicationFixtures(t testing.TB, repositoryRoot string) []applicationFixture {
	t.Helper()

	fixturesRoot := filepath.Join(repositoryRoot, "testfiles", "applications")
	entries, err := os.ReadDir(fixturesRoot)
	if err != nil {
		t.Fatalf("reading application fixtures at %s: %v", fixturesRoot, err)
	}

	var fixtures []applicationFixture
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		fixtures = append(fixtures, loadApplicationFixture(t, filepath.Join(fixturesRoot, entry.Name()), entry.Name()))
	}
	if len(fixtures) == 0 {
		t.Fatalf("no application fixtures found under %s", fixturesRoot)
	}
	sort.Slice(fixtures, func(i, j int) bool { return fixtures[i].name < fixtures[j].name })
	return fixtures
}

func loadApplicationFixture(t testing.TB, fixtureRoot, name string) applicationFixture {
	t.Helper()

	metadataPath := filepath.Join(fixtureRoot, "fixture.json")
	metadata, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("application fixture %q is missing readable fixture.json: %v", name, err)
	}

	var config applicationFixtureConfig
	decoder := json.NewDecoder(bytes.NewReader(metadata))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		t.Fatalf("decoding %s: %v", metadataPath, err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		t.Fatalf("decoding %s: %v", metadataPath, err)
	}
	validateApplicationFixtureConfig(t, metadataPath, config)

	expectedStdoutPath := filepath.Join(fixtureRoot, "expected.stdout")
	expectedStdout, err := os.ReadFile(expectedStdoutPath)
	if err != nil {
		t.Fatalf("application fixture %q is missing readable expected.stdout: %v", name, err)
	}

	// stderr is normally empty. A fixture that intentionally writes to stderr can
	// opt into an exact snapshot without complicating the common metadata format.
	expectedStderrPath := filepath.Join(fixtureRoot, "expected.stderr")
	expectedStderr, err := os.ReadFile(expectedStderrPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("reading %s: %v", expectedStderrPath, err)
	}

	sourceRoot := filepath.Join(fixtureRoot, "src")
	if config.SourceRoot != "" {
		if config.SourceRoot != strings.TrimSpace(config.SourceRoot) || strings.Contains(config.SourceRoot, "\\") || filepath.IsAbs(config.SourceRoot) || path.Clean(config.SourceRoot) != config.SourceRoot {
			t.Fatalf("%s: source_root must be a trimmed, relative slash-form path, got %q", metadataPath, config.SourceRoot)
		}
		sourceRoot = filepath.Join(fixtureRoot, filepath.FromSlash(config.SourceRoot))
	}
	sourceRoot = filepath.Clean(sourceRoot)
	testfilesRoot := filepath.Clean(filepath.Join(fixtureRoot, "..", ".."))
	relativeSourceRoot, err := filepath.Rel(testfilesRoot, sourceRoot)
	if err != nil || relativeSourceRoot == ".." || strings.HasPrefix(relativeSourceRoot, ".."+string(filepath.Separator)) {
		t.Fatalf("%s: source_root must resolve inside %s", metadataPath, testfilesRoot)
	}
	info, err := os.Stat(sourceRoot)
	if err != nil || !info.IsDir() {
		t.Fatalf("application fixture %q source root %s must be a readable directory", name, sourceRoot)
	}

	return applicationFixture{
		name:           name,
		root:           fixtureRoot,
		sourceRoot:     sourceRoot,
		expectedStdout: expectedStdout,
		expectedStderr: expectedStderr,
		config:         config,
	}
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("fixture.json must contain exactly one JSON object")
		}
		return err
	}
	return nil
}

func validateApplicationFixtureConfig(t testing.TB, metadataPath string, config applicationFixtureConfig) {
	t.Helper()

	requireTrimmed := func(field, value string) {
		if value == "" || value != strings.TrimSpace(value) {
			t.Fatalf("%s: %s must be nonempty and have no surrounding whitespace", metadataPath, field)
		}
	}
	requireTrimmed("main_class", config.MainClass)
	requireTrimmed("module_path", config.ModulePath)
	requireTrimmed("status", config.Status)

	mainParts := strings.Split(config.MainClass, ".")
	if len(mainParts) < 2 {
		t.Fatalf("%s: main_class must be fully qualified, got %q", metadataPath, config.MainClass)
	}
	for _, part := range mainParts {
		if !javaIdentifier.MatchString(part) {
			t.Fatalf("%s: main_class %q contains invalid Java identifier %q", metadataPath, config.MainClass, part)
		}
	}

	if strings.Contains(config.ModulePath, "\\") || path.Clean(config.ModulePath) != config.ModulePath || strings.HasPrefix(config.ModulePath, "/") {
		t.Fatalf("%s: module_path must be a clean relative slash-form Java package prefix, got %q", metadataPath, config.ModulePath)
	}
	moduleParts := strings.Split(config.ModulePath, "/")
	for _, part := range moduleParts {
		if !javaIdentifier.MatchString(part) {
			t.Fatalf("%s: module_path %q contains invalid package segment %q", metadataPath, config.ModulePath, part)
		}
	}

	mainPackage := strings.Join(mainParts[:len(mainParts)-1], ".")
	modulePackage := strings.Join(moduleParts, ".")
	if mainPackage != modulePackage && !strings.HasPrefix(mainPackage, modulePackage+".") {
		t.Fatalf("%s: main_class package %q is outside module_path package prefix %q", metadataPath, mainPackage, modulePackage)
	}

	switch config.Status {
	case "passing":
		if config.KnownGap != "" || config.ExpectedFailureStage != "" || config.ExpectedFailureContains != "" {
			t.Fatalf("%s: passing fixtures must omit known_gap, expected_failure_stage, and expected_failure_contains", metadataPath)
		}
	case "known_gap":
		requireTrimmed("known_gap", config.KnownGap)
		requireTrimmed("expected_failure_stage", config.ExpectedFailureStage)
		requireTrimmed("expected_failure_contains", config.ExpectedFailureContains)
		switch config.ExpectedFailureStage {
		case "transpile", "go_compile", "go_run", "output":
		default:
			t.Fatalf("%s: expected_failure_stage must be transpile, go_compile, go_run, or output; got %q", metadataPath, config.ExpectedFailureStage)
		}
	default:
		t.Fatalf("%s: status must be %q or %q, got %q", metadataPath, "passing", "known_gap", config.Status)
	}
}

func runApplicationFixture(t *testing.T, repositoryRoot string, fixture applicationFixture) *applicationParityFailure {
	t.Helper()

	javaSources := collectApplicationJavaSources(t, fixture.sourceRoot)
	testRoot := t.TempDir()
	javaClasses := filepath.Join(testRoot, "java-classes")
	javaClasspath := filepath.Join(testRoot, "java-classpath")
	javaWork := filepath.Join(testRoot, "java-work")
	goOutput := filepath.Join(testRoot, "go-output")
	goWork := filepath.Join(testRoot, "go-work")
	goCache := filepath.Join(testRoot, "go-cache")
	for _, directory := range []string{javaClasses, javaClasspath, javaWork, goWork, goCache} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("creating fixture build directory %s: %v", directory, err)
		}
	}

	javaCompileArgs := append([]string{
		"--release", applicationJavaRelease,
		"-encoding", "UTF-8",
		"-classpath", javaClasspath,
		"-d", javaClasses,
	}, javaSources...)
	javaCompile := runApplicationCommand(applicationFixtureTimeout, fixture.root, deterministicApplicationEnv(nil), "javac", javaCompileArgs...)
	requireApplicationCommandStarted(t, "javac", javaCompile)
	if javaCompile.timedOut || javaCompile.exitCode != 0 {
		t.Fatalf("Java oracle did not compile (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
			javaCompile.exitCode, javaCompile.timedOut, javaCompile.stdout, javaCompile.stderr)
	}

	javaArgs := []string{
		"-Dfile.encoding=UTF-8",
		"-Duser.language=en",
		"-Duser.country=US",
		"-Duser.timezone=UTC",
		"-cp", javaClasses,
		fixture.config.MainClass,
	}
	javaResult := runApplicationCommand(applicationFixtureTimeout, javaWork, deterministicApplicationEnv(nil), "java", javaArgs...)
	requireApplicationCommandStarted(t, "java", javaResult)
	if javaResult.timedOut || javaResult.exitCode != 0 {
		t.Fatalf("Java oracle did not exit successfully (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
			javaResult.exitCode, javaResult.timedOut, javaResult.stdout, javaResult.stderr)
	}
	if !bytes.Equal(javaResult.stdout, fixture.expectedStdout) {
		t.Fatalf("Java oracle no longer matches expected.stdout; update the fixture only if this behavior change is intentional\n%s",
			formatApplicationOutputDifference("expected.stdout", fixture.expectedStdout, "java stdout", javaResult.stdout))
	}
	if !bytes.Equal(javaResult.stderr, fixture.expectedStderr) {
		t.Fatalf("Java oracle stderr does not match expected.stderr (stderr must be empty when that file is absent)\n%s",
			formatApplicationOutputDifference("expected stderr", fixture.expectedStderr, "java stderr", javaResult.stderr))
	}

	if err := runApplicationTranspiler(fixture, goOutput); err != nil {
		return &applicationParityFailure{stage: "transpile", detail: err.Error()}
	}
	if err := configureGeneratedApplicationModule(repositoryRoot, goOutput); err != nil {
		t.Fatalf("configuring generated module: %v", err)
	}

	goBinary := filepath.Join(testRoot, "generated-application")
	goEnv := deterministicApplicationEnv(map[string]string{
		"GOCACHE": goCache,
		"GOWORK":  "off",
		"GOFLAGS": "",
	})
	// Compile every converted package, including sources not reachable from main.
	// javac validated the entire Java tree, so parity should hold that same bar.
	goModuleCompile := runApplicationCommand(applicationFixtureTimeout, goOutput, goEnv, "go", "build", "-mod=mod", "./...")
	requireApplicationCommandStarted(t, "go build ./...", goModuleCompile)
	if goModuleCompile.timedOut || goModuleCompile.exitCode != 0 {
		return &applicationParityFailure{
			stage: "go_compile",
			detail: fmt.Sprintf("generated Go module did not compile (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
				goModuleCompile.exitCode, goModuleCompile.timedOut, goModuleCompile.stdout, goModuleCompile.stderr),
		}
	}

	if err := writeApplicationDriver(fixture.config, goOutput); err != nil {
		t.Fatalf("writing generated application driver: %v", err)
	}
	goCompile := runApplicationCommand(applicationFixtureTimeout, goOutput, goEnv, "go", "build", "-mod=mod", "-o", goBinary, "./paritydriver")
	requireApplicationCommandStarted(t, "go build", goCompile)
	if goCompile.timedOut || goCompile.exitCode != 0 {
		return &applicationParityFailure{
			stage: "go_compile",
			detail: fmt.Sprintf("generated Go did not compile (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
				goCompile.exitCode, goCompile.timedOut, goCompile.stdout, goCompile.stderr),
		}
	}

	goResult := runApplicationCommand(applicationFixtureTimeout, goWork, deterministicApplicationEnv(nil), goBinary)
	requireApplicationCommandStarted(t, "generated Go application", goResult)
	if goResult.timedOut || goResult.exitCode != javaResult.exitCode {
		return &applicationParityFailure{
			stage: "go_run",
			detail: fmt.Sprintf("runtime exit mismatch (Java=%d, Go=%d, Go timeout=%t)\nGo stdout:\n%s\nGo stderr:\n%s",
				javaResult.exitCode, goResult.exitCode, goResult.timedOut, goResult.stdout, goResult.stderr),
		}
	}
	if !bytes.Equal(goResult.stdout, javaResult.stdout) || !bytes.Equal(goResult.stderr, javaResult.stderr) {
		detail := formatApplicationOutputDifference("java stdout", javaResult.stdout, "go stdout", goResult.stdout)
		if !bytes.Equal(goResult.stderr, javaResult.stderr) {
			detail += "\n" + formatApplicationOutputDifference("java stderr", javaResult.stderr, "go stderr", goResult.stderr)
		}
		return &applicationParityFailure{stage: "output", detail: detail}
	}

	return nil
}

func collectApplicationJavaSources(t testing.TB, sourceRoot string) []string {
	t.Helper()

	var sources []string
	err := filepath.WalkDir(sourceRoot, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".java" {
			sources = append(sources, filePath)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("collecting Java sources from %s: %v", sourceRoot, err)
	}
	if len(sources) == 0 {
		t.Fatalf("application fixture contains no Java sources under %s", sourceRoot)
	}
	sort.Strings(sources)
	return sources
}

func runApplicationTranspiler(fixture applicationFixture, outputRoot string) (err error) {
	previousGlobal := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	defer func() {
		symbol.GlobalScope = previousGlobal
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("java2go panicked while converting %s: %v", fixture.name, recovered)
		}
	}()

	args := []string{
		"-w",
		"-sync",
		"-strict",
		"-init-go-mod",
		"-module", fixture.config.ModulePath,
		"-output", outputRoot,
		fixture.sourceRoot,
	}
	if runErr := java2go.Run(args); runErr != nil {
		return fmt.Errorf("java2go failed while converting %s: %w", fixture.name, runErr)
	}
	return nil
}

func configureGeneratedApplicationModule(repositoryRoot, outputRoot string) error {
	goModPath := filepath.Join(outputRoot, "go.mod")
	goMod, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("reading generated go.mod: %w", err)
	}

	repositoryPath := strconv.Quote(filepath.ToSlash(repositoryRoot))
	goMod = append(goMod, []byte(fmt.Sprintf("\nrequire %s v0.0.0\n\nreplace %s => %s\n",
		java2goModulePath, java2goModulePath, repositoryPath))...)
	if err := os.WriteFile(goModPath, goMod, 0o644); err != nil {
		return fmt.Errorf("updating generated go.mod: %w", err)
	}
	return nil
}

func writeApplicationDriver(config applicationFixtureConfig, outputRoot string) error {
	mainClassParts := strings.Split(config.MainClass, ".")
	mainPackageParts := mainClassParts[:len(mainClassParts)-1]
	moduleParts := strings.Split(config.ModulePath, "/")
	relativePackage := mainPackageParts[len(moduleParts):]
	mainImport := config.ModulePath
	if len(relativePackage) > 0 {
		mainImport += "/" + strings.Join(relativePackage, "/")
	}

	driverRoot := filepath.Join(outputRoot, "paritydriver")
	if _, err := os.Stat(driverRoot); err == nil {
		return fmt.Errorf("generated source collides with reserved paritydriver directory")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking paritydriver directory: %w", err)
	}
	if err := os.MkdirAll(driverRoot, 0o755); err != nil {
		return err
	}

	driver := fmt.Sprintf("package main\n\nimport application %q\n\nfunc main() { application.Main() }\n", mainImport)
	if err := os.WriteFile(filepath.Join(driverRoot, "main.go"), []byte(driver), 0o644); err != nil {
		return err
	}
	return nil
}

func runApplicationCommand(timeout time.Duration, workingDirectory string, environment []string, name string, args ...string) applicationCommandResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = workingDirectory
	command.Env = environment
	command.Stdout = &stdout
	command.Stderr = &stderr

	result := applicationCommandResult{exitCode: 0}
	err := command.Run()
	result.stdout = append([]byte(nil), stdout.Bytes()...)
	result.stderr = append([]byte(nil), stderr.Bytes()...)
	result.timedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
	if err == nil {
		return result
	}

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.exitCode = exitError.ExitCode()
		return result
	}
	result.exitCode = -1
	result.err = err
	return result
}

func deterministicApplicationEnv(overrides map[string]string) []string {
	values := make(map[string]string)
	for _, pair := range os.Environ() {
		key, value, ok := strings.Cut(pair, "=")
		if ok {
			values[key] = value
		}
	}

	// User-level JVM hooks can alter behavior and write banners to stderr.
	delete(values, "JAVA_TOOL_OPTIONS")
	delete(values, "JDK_JAVA_OPTIONS")
	delete(values, "JDK_JAVAC_OPTIONS")
	delete(values, "_JAVA_OPTIONS")
	delete(values, "CLASSPATH")
	// Host-level Go runtime tuning would make parity runs and comparative
	// benchmarks depend on the invoking shell. Both generated binaries therefore
	// run with Go's documented defaults unless a benchmark explicitly overrides a
	// setting here.
	delete(values, "GOGC")
	delete(values, "GOMEMLIMIT")
	delete(values, "GODEBUG")
	delete(values, "GOMAXPROCS")
	values["LANG"] = "C"
	values["LC_ALL"] = "C"
	values["TZ"] = "UTC"
	for key, value := range overrides {
		values[key] = value
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	environment := make([]string, 0, len(keys))
	for _, key := range keys {
		environment = append(environment, key+"="+values[key])
	}
	return environment
}

func requireApplicationTool(t testing.TB, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Fatalf("application parity requires %s on PATH: %v", name, err)
	}
}

func requireApplicationCommandStarted(t testing.TB, description string, result applicationCommandResult) {
	t.Helper()
	if result.err != nil {
		t.Fatalf("starting %s: %v", description, result.err)
	}
}

func formatApplicationOutputDifference(wantName string, want []byte, gotName string, got []byte) string {
	return fmt.Sprintf("--- %s (%d bytes) ---\n%s\n--- %s (%d bytes) ---\n%s\n--- quoted %s ---\n%q\n--- quoted %s ---\n%q",
		wantName, len(want), want, gotName, len(got), got, wantName, want, gotName, got)
}
