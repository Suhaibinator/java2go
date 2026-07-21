package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// BenchmarkApplicationPerformance compares complete Java applications with
// their transpiled Go counterparts. Build and transpilation happen before the
// timer starts. Each measured operation is a batch of fresh processes, so the
// result deliberately includes runtime startup and shutdown as well as the
// application workload.
//
// Run one measured batch per runtime with:
//
//	go test ./e2e -run '^$' -bench '^BenchmarkApplicationPerformance$' -benchtime=1x
//
// Use -count=N for independent samples. Every warm-up and measured execution
// must still produce the fixture's exact parity oracle.
func BenchmarkApplicationPerformance(b *testing.B) {
	repositoryRoot := moduleRoot(b)
	requireApplicationTool(b, "javac")
	requireApplicationTool(b, "java")
	requireApplicationTool(b, "go")

	fixtures := discoverApplicationFixtures(b, repositoryRoot)
	benchmarks := discoverApplicationBenchmarks(b, fixtures)
	if len(benchmarks) == 0 {
		b.Fatalf("no benchmark.json files found under testfiles/applications")
	}

	for _, benchmark := range benchmarks {
		benchmark := benchmark
		b.Run(benchmark.fixture.name, func(b *testing.B) {
			prepared := prepareApplicationBenchmark(b, repositoryRoot, benchmark)

			b.Run("java", func(b *testing.B) {
				runPreparedApplicationBenchmark(b, benchmark, prepared.javaCommand)
			})
			b.Run("go", func(b *testing.B) {
				runPreparedApplicationBenchmark(b, benchmark, prepared.goCommand)
			})
		})
	}
}

type applicationBenchmarkConfig struct {
	Category    string `json:"category"`
	Description string `json:"description"`
	Iterations  int    `json:"iterations"`
}

type applicationBenchmark struct {
	fixture applicationFixture
	config  applicationBenchmarkConfig
}

type preparedBenchmarkCommand struct {
	description      string
	workingDirectory string
	environment      []string
	name             string
	args             []string
}

type preparedApplicationBenchmark struct {
	javaCommand preparedBenchmarkCommand
	goCommand   preparedBenchmarkCommand
}

var applicationBenchmarkCategory = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func TestApplicationBenchmarkMetadata(t *testing.T) {
	fixtures := discoverApplicationFixtures(t, moduleRoot(t))
	if benchmarks := discoverApplicationBenchmarks(t, fixtures); len(benchmarks) == 0 {
		t.Fatal("no benchmark.json files found under testfiles/applications")
	}
}

func discoverApplicationBenchmarks(b testing.TB, fixtures []applicationFixture) []applicationBenchmark {
	b.Helper()

	benchmarks := make([]applicationBenchmark, 0, len(fixtures))
	for _, fixture := range fixtures {
		metadataPath := filepath.Join(fixture.root, "benchmark.json")
		metadata, err := os.ReadFile(metadataPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			b.Fatalf("reading %s: %v", metadataPath, err)
		}

		var config applicationBenchmarkConfig
		decoder := json.NewDecoder(bytes.NewReader(metadata))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&config); err != nil {
			b.Fatalf("decoding %s: %v", metadataPath, err)
		}
		if err := ensureJSONEOF(decoder); err != nil {
			b.Fatalf("decoding %s: %v", metadataPath, err)
		}
		if !applicationBenchmarkCategory.MatchString(config.Category) {
			b.Fatalf("%s: category must match %s, got %q", metadataPath, applicationBenchmarkCategory, config.Category)
		}
		if config.Description == "" || config.Description != string(bytes.TrimSpace([]byte(config.Description))) {
			b.Fatalf("%s: description must be nonempty and have no surrounding whitespace", metadataPath)
		}
		if config.Iterations < 1 || config.Iterations > 100 {
			b.Fatalf("%s: iterations must be between 1 and 100, got %d", metadataPath, config.Iterations)
		}
		if fixture.config.Status != "passing" {
			b.Fatalf("%s: benchmark fixtures must have passing parity status, got %q", metadataPath, fixture.config.Status)
		}

		benchmarks = append(benchmarks, applicationBenchmark{fixture: fixture, config: config})
	}
	return benchmarks
}

func prepareApplicationBenchmark(b testing.TB, repositoryRoot string, benchmark applicationBenchmark) preparedApplicationBenchmark {
	b.Helper()

	fixture := benchmark.fixture
	javaSources := collectApplicationJavaSources(b, fixture.sourceRoot)
	testRoot := b.TempDir()
	javaClasses := filepath.Join(testRoot, "java-classes")
	javaClasspath := filepath.Join(testRoot, "java-classpath")
	javaWork := filepath.Join(testRoot, "java-work")
	goOutput := filepath.Join(testRoot, "go-output")
	goWork := filepath.Join(testRoot, "go-work")
	goCache := filepath.Join(testRoot, "go-cache")
	for _, directory := range []string{javaClasses, javaClasspath, javaWork, goWork, goCache} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			b.Fatalf("creating benchmark build directory %s: %v", directory, err)
		}
	}

	javaCompileArgs := append([]string{
		"--release", applicationJavaRelease,
		"-encoding", "UTF-8",
		"-classpath", javaClasspath,
		"-d", javaClasses,
	}, javaSources...)
	javaCompile := runApplicationCommand(applicationFixtureTimeout, fixture.root, deterministicApplicationEnv(nil), "javac", javaCompileArgs...)
	requireApplicationCommandStarted(b, "javac", javaCompile)
	if javaCompile.timedOut || javaCompile.exitCode != 0 {
		b.Fatalf("Java benchmark did not compile (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
			javaCompile.exitCode, javaCompile.timedOut, javaCompile.stdout, javaCompile.stderr)
	}

	if err := runApplicationTranspiler(fixture, goOutput); err != nil {
		b.Fatalf("transpiling benchmark: %v", err)
	}
	if err := configureGeneratedApplicationModule(repositoryRoot, goOutput); err != nil {
		b.Fatalf("configuring generated benchmark module: %v", err)
	}

	goEnvironment := deterministicApplicationEnv(map[string]string{
		"GOCACHE": goCache,
		"GOWORK":  "off",
		"GOFLAGS": "",
	})
	goModuleCompile := runApplicationCommand(applicationFixtureTimeout, goOutput, goEnvironment, "go", "build", "-mod=mod", "./...")
	requireApplicationCommandStarted(b, "go build ./...", goModuleCompile)
	if goModuleCompile.timedOut || goModuleCompile.exitCode != 0 {
		b.Fatalf("generated benchmark module did not compile (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
			goModuleCompile.exitCode, goModuleCompile.timedOut, goModuleCompile.stdout, goModuleCompile.stderr)
	}

	if err := writeApplicationDriver(fixture.config, goOutput); err != nil {
		b.Fatalf("writing generated benchmark driver: %v", err)
	}
	goBinary := filepath.Join(testRoot, "generated-benchmark")
	goCompile := runApplicationCommand(applicationFixtureTimeout, goOutput, goEnvironment, "go", "build", "-mod=mod", "-o", goBinary, "./paritydriver")
	requireApplicationCommandStarted(b, "go build", goCompile)
	if goCompile.timedOut || goCompile.exitCode != 0 {
		b.Fatalf("generated benchmark did not compile (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
			goCompile.exitCode, goCompile.timedOut, goCompile.stdout, goCompile.stderr)
	}

	return preparedApplicationBenchmark{
		javaCommand: preparedBenchmarkCommand{
			description:      "Java benchmark",
			workingDirectory: javaWork,
			environment:      deterministicApplicationEnv(nil),
			name:             "java",
			args: []string{
				"-Dfile.encoding=UTF-8",
				"-Duser.language=en",
				"-Duser.country=US",
				"-Duser.timezone=UTC",
				"-cp", javaClasses,
				fixture.config.MainClass,
			},
		},
		goCommand: preparedBenchmarkCommand{
			description:      "generated Go benchmark",
			workingDirectory: goWork,
			environment:      deterministicApplicationEnv(nil),
			name:             goBinary,
		},
	}
}

func runPreparedApplicationBenchmark(b *testing.B, benchmark applicationBenchmark, command preparedBenchmarkCommand) {
	b.Helper()

	// One untimed execution catches invalid output and gives managed runtimes a
	// consistent filesystem/cache warm-up before measurements begin.
	validateApplicationBenchmarkRun(b, benchmark.fixture, command)
	b.ResetTimer()
	for operation := 0; operation < b.N; operation++ {
		for iteration := 0; iteration < benchmark.config.Iterations; iteration++ {
			validateApplicationBenchmarkRun(b, benchmark.fixture, command)
		}
	}
	b.StopTimer()

	runs := b.N * benchmark.config.Iterations
	if runs > 0 {
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(runs), "ns/run")
	}
}

func validateApplicationBenchmarkRun(b testing.TB, fixture applicationFixture, command preparedBenchmarkCommand) {
	b.Helper()

	result := runApplicationCommand(applicationFixtureTimeout, command.workingDirectory, command.environment, command.name, command.args...)
	requireApplicationCommandStarted(b, command.description, result)
	if result.timedOut || result.exitCode != 0 {
		b.Fatalf("%s failed (exit %d, timeout=%t)\nstdout:\n%s\nstderr:\n%s",
			command.description, result.exitCode, result.timedOut, result.stdout, result.stderr)
	}
	if !bytes.Equal(result.stdout, fixture.expectedStdout) || !bytes.Equal(result.stderr, fixture.expectedStderr) {
		detail := formatApplicationOutputDifference("expected stdout", fixture.expectedStdout, command.description+" stdout", result.stdout)
		if !bytes.Equal(result.stderr, fixture.expectedStderr) {
			detail += "\n" + formatApplicationOutputDifference("expected stderr", fixture.expectedStderr, command.description+" stderr", result.stderr)
		}
		b.Fatalf("%s violated the parity oracle during benchmarking:\n%s", command.description, detail)
	}
}

func ExampleBenchmarkApplicationPerformance() {
	fmt.Println("go test ./e2e -run ^$ -bench ^BenchmarkApplicationPerformance$ -benchtime=1x -count=5")
	// Output: go test ./e2e -run ^$ -bench ^BenchmarkApplicationPerformance$ -benchtime=1x -count=5
}
