package campaign

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

const DefaultJDK = ""

type Config struct {
	Repository   string
	Fixture      string
	Artifacts    string
	JDK          string
	Transpiler   string
	BuildTimeout time.Duration
	RunTimeout   time.Duration
	Race         bool
	StressRuns   int
}
type Failure struct {
	Stage  string `json:"stage"`
	Detail string `json:"detail"`
}
type Observation struct {
	Seed   int        `json:"seed"`
	Repeat int        `json:"repeat"`
	Java   Execution  `json:"java"`
	Go     *Execution `json:"go,omitempty"`
}
type Report struct {
	RaceEnabled         bool                 `json:"race_enabled"`
	StressRuns          int                  `json:"stress_runs"`
	StressObservations  []Observation        `json:"stress_observations,omitempty"`
	TranspilerSHA256    string               `json:"transpiler_sha256,omitempty"`
	Implementation      Implementation       `json:"implementation"`
	ImplementationAfter *Implementation      `json:"implementation_after,omitempty"`
	PriorFailure        *Failure             `json:"prior_failure,omitempty"`
	JDK                 string               `json:"jdk"`
	MavenLockSHA256     string               `json:"maven_lock_sha256"`
	BuildSemantics      string               `json:"build_semantics"`
	Name                string               `json:"name"`
	Passed              bool                 `json:"passed"`
	ArtifactDirectory   string               `json:"artifact_directory"`
	ManifestSHA256      string               `json:"manifest_sha256"`
	LockSHA256          string               `json:"lock_sha256"`
	SourceHashes        map[string]string    `json:"source_hashes"`
	DependencyMode      string               `json:"dependency_mode"`
	Failure             *Failure             `json:"failure,omitempty"`
	Stages              map[string]Execution `json:"stages"`
	Observations        []Observation        `json:"observations"`
}
type prepared struct {
	appSources, allSources, classpath []string
	project                           string
	mappings                          []string
	resources                         map[string][]byte
}

// Run always writes a report and retains all artifacts. A report only passes
// after every declared class builds, all isolated execution pairs match, and
// any requested race-enabled stress executions match their validated JVM oracle.
func Run(ctx context.Context, c Config) (report Report, err error) {
	if c.StressRuns < 0 {
		return report, fmt.Errorf("stress runs must be nonnegative")
	}
	c.Repository, err = filepath.Abs(c.Repository)
	if err != nil {
		return report, err
	}
	c.Fixture, err = filepath.Abs(c.Fixture)
	if err != nil {
		return report, err
	}
	if c.BuildTimeout == 0 {
		c.BuildTimeout = 5 * time.Minute
	}
	if c.RunTimeout == 0 {
		c.RunTimeout = time.Minute
	}
	if c.BuildTimeout <= 0 || c.RunTimeout <= 0 {
		return report, fmt.Errorf("timeouts must be positive")
	}
	if c.Artifacts == "" {
		c.Artifacts = filepath.Join(c.Repository, ".campaign/runs")
	}
	c.Artifacts, err = filepath.Abs(c.Artifacts)
	if err != nil {
		return report, err
	}
	if c.Transpiler != "" {
		c.Transpiler, err = filepath.Abs(c.Transpiler)
		if err != nil {
			return report, err
		}
	}
	if err = os.MkdirAll(c.Artifacts, 0755); err != nil {
		return report, err
	}
	runDir, e := os.MkdirTemp(c.Artifacts, time.Now().UTC().Format("20060102T150405Z")+"-")
	if e != nil {
		return report, e
	}
	report = Report{RaceEnabled: c.Race, StressRuns: c.StressRuns, ArtifactDirectory: runDir, SourceHashes: map[string]string{}, Stages: map[string]Execution{}, DependencyMode: "frozen selected whole-class source closure"}
	defer func() {
		if e := writeJSON(filepath.Join(runDir, "report.json"), report); err == nil && e != nil {
			err = e
		}
	}()
	fail := func(stage, detail string) (Report, error) {
		report.Failure = &Failure{stage, detail}
		return report, fmt.Errorf("%s: %s (artifacts %s)", stage, detail, runDir)
	}
	implementation, e := implementationFingerprint(c.Repository)
	if e != nil {
		return fail("implementation-freeze", e.Error())
	}
	report.Implementation = implementation
	defer func() {
		after, checkErr := implementationFingerprint(c.Repository)
		report.ImplementationAfter = &after
		if checkErr != nil || implementation.SHA256 != after.SHA256 {
			report.Passed = false
			report.PriorFailure = report.Failure
			report.Failure = &Failure{"implementation-changed", "compiler/runtime/harness changed during run; rerun stable inputs"}
			err = fmt.Errorf("implementation changed during run")
		}
	}()
	var jdkResult Execution
	c.JDK, jdkResult, e = resolveJDK(ctx, c.JDK)
	if e != nil {
		return fail("java-toolchain", e.Error())
	}
	report.JDK = c.JDK
	report.Stages["java-toolchain"] = jdkResult
	initialHashes, e := fixtureHashes(c.Fixture)
	if e != nil {
		return fail("freeze", e.Error())
	}
	defer func() {
		finalHashes, checkErr := fixtureHashes(c.Fixture)
		if checkErr != nil || !reflect.DeepEqual(initialHashes, finalHashes) {
			report.Passed = false
			report.Failure = &Failure{"freeze-changed", "fixture/oracle inputs changed during run"}
			err = fmt.Errorf("fixture/oracle inputs changed during run")
		}
	}()
	m, e := LoadManifest(c.Fixture)
	if e != nil {
		return fail("manifest", e.Error())
	}
	report.Name = m.Name
	manifestBytes, e := os.ReadFile(filepath.Join(c.Fixture, "fixture.json"))
	if e != nil {
		return fail("manifest", e.Error())
	}
	report.ManifestSHA256 = hash(manifestBytes)
	if e = write(filepath.Join(runDir, "frozen/fixture.json"), manifestBytes); e != nil {
		return fail("freeze", e.Error())
	}
	lock, e := LoadLock(c.Repository)
	if e != nil {
		return fail("dependency-lock", e.Error())
	}
	defer func() {
		if checkErr := lock.Verify(c.Repository); checkErr != nil {
			report.Passed = false
			report.PriorFailure = report.Failure
			report.Failure = &Failure{"dependency-lock-changed", checkErr.Error()}
			err = checkErr
		}
	}()
	lockBytes, e := os.ReadFile(filepath.Join(c.Repository, "campaign/dependencies.lock.json"))
	if e != nil {
		return fail("dependency-lock", e.Error())
	}
	report.LockSHA256 = hash(lockBytes)
	if e = write(filepath.Join(runDir, "frozen/dependencies.lock.json"), lockBytes); e != nil {
		return fail("freeze", e.Error())
	}
	expected, e := loadExpected(c.Fixture, m)
	if e != nil {
		return fail("oracle-freeze", e.Error())
	}
	if e = writeJSON(filepath.Join(runDir, "frozen/fixture-hashes.json"), initialHashes); e != nil {
		return fail("freeze", e.Error())
	}
	if e = writeJSON(filepath.Join(runDir, "frozen/expected.json"), expected); e != nil {
		return fail("freeze", e.Error())
	}
	p, e := prepare(c, m, lock, runDir, &report)
	if e != nil {
		return fail("dependency-resolution", e.Error())
	}
	if e = writeJSON(filepath.Join(runDir, "frozen/source-hashes.json"), report.SourceHashes); e != nil {
		return fail("freeze", e.Error())
	}
	stage := func(name, dir string, command []string) bool {
		r := execute(ctx, c.BuildTimeout, dir, command)
		report.Stages[name] = r
		_ = writeJSON(filepath.Join(runDir, "stages", name+".json"), r)
		_ = write(filepath.Join(runDir, "stages", name+".stdout"), []byte(r.Stdout))
		_ = write(filepath.Join(runDir, "stages", name+".stderr"), []byte(r.Stderr))
		return r.ExitCode == 0 && !r.TimedOut && r.Error == ""
	}
	stageDetail := func(name string) string {
		r := report.Stages[name]
		if r.TimedOut {
			return "build timed out"
		}
		return fmt.Sprintf("exit=%d error=%s\n%s\n%s", r.ExitCode, r.Error, r.Stdout, r.Stderr)
	}
	report.MavenLockSHA256, e = verifyMavenLock(c.Repository)
	if e != nil {
		return fail("maven-resolution", e.Error())
	}
	mavenLockBytes, e := os.ReadFile(filepath.Join(c.Repository, "campaign/maven.lock.json"))
	if e != nil {
		return fail("maven-resolution", e.Error())
	}
	if e = write(filepath.Join(runDir, "frozen/maven.lock.json"), mavenLockBytes); e != nil {
		return fail("freeze", e.Error())
	}
	defer func() {
		after, checkErr := verifyMavenLock(c.Repository)
		if checkErr != nil || after != report.MavenLockSHA256 {
			report.Passed = false
			report.PriorFailure = report.Failure
			report.Failure = &Failure{"maven-lock-changed", "maven build dependencies changed during run"}
			err = fmt.Errorf("maven build dependencies changed during run")
		}
	}()
	mavenProject := filepath.Join(runDir, "frozen/maven-project")
	if e = copyMavenProject(c.Fixture, mavenProject); e != nil {
		return fail("maven-build", e.Error())
	}
	report.BuildSemantics = "original POM validated with offline Maven compile; transpiler consumes explicit source-only ingestion POM, not original Maven build semantics"
	maven := filepath.Join(c.Repository, ".campaign/tools/apache-maven-3.9.16/bin/mvn")
	if !stage("maven-build", mavenProject, []string{"env", "JAVA_HOME=" + c.JDK, maven, "--batch-mode", "--offline", "-Dmaven.repo.local=" + filepath.Join(c.Repository, ".campaign/m2"), "-Dmaven.compiler.encoding=UTF-8", "-f", filepath.Join(mavenProject, m.POM), "compile"}) {
		return fail(classifyMavenFailure(report.Stages["maven-build"]), stageDetail("maven-build"))
	}
	javac := filepath.Join(c.JDK, "bin/javac")
	java := filepath.Join(c.JDK, "bin/java")
	closureClasses := filepath.Join(runDir, "java-source-closure")
	javaClasses := filepath.Join(runDir, "java-oracle")
	for _, d := range []string{closureClasses, javaClasses} {
		if e = os.MkdirAll(d, 0755); e != nil {
			return fail("java-compile", e.Error())
		}
	}
	// No dependency binary is on this classpath: javac must resolve the entire
	// declared source graph, including unused methods and optional runtime paths.
	args := append([]string{javac, "--release", "21", "-encoding", "UTF-8", "-proc:none", "-implicit:none", "-classpath", closureClasses, "-d", closureClasses}, p.allSources...)
	if !stage("dependency-source-compile", runDir, args) {
		return fail("dependency-source-compile", stageDetail("dependency-source-compile"))
	}
	args = append([]string{javac, "--release", "21", "-encoding", "UTF-8", "-proc:none", "-classpath", strings.Join(p.classpath, string(os.PathListSeparator)), "-d", javaClasses}, p.appSources...)
	if !stage("java-oracle-compile", runDir, args) {
		return fail("java-oracle-compile", stageDetail("java-oracle-compile"))
	}
	for target, b := range p.resources {
		if e = write(filepath.Join(javaClasses, target), b); e != nil {
			return fail("resources", e.Error())
		}
	}
	for _, seed := range m.Seeds {
		var baseline *Execution
		for repeat := 1; repeat <= m.Repeats; repeat++ {
			dir := filepath.Join(runDir, "runs", fmt.Sprintf("seed-%d-repeat-%d", seed, repeat), "java")
			if e = seedResources(dir, p.resources); e != nil {
				return fail("resources", e.Error())
			}
			args = append([]string{java, "-Dfile.encoding=UTF-8", "-Duser.language=en", "-Duser.country=US", "-Duser.timezone=UTC", "-cp", strings.Join(append([]string{javaClasses}, p.classpath...), string(os.PathListSeparator)), m.MainClass}, seedArgs(m.Args, seed)...)
			result := execute(ctx, c.RunTimeout, dir, args)
			if e = saveExecution(filepath.Join(dir, "..", "java"), result); e != nil {
				return fail("artifacts", e.Error())
			}
			result.Files, e = outputFiles(dir, m.OutputFiles, p.resources)
			if e != nil {
				return fail("java-output-files", e.Error())
			}
			report.Observations = append(report.Observations, Observation{Seed: seed, Repeat: repeat, Java: result})
			if diff := compareExpected(expected[seed], result); diff != "" {
				return fail("java-oracle-changed", fmt.Sprintf("seed %d repeat %d: %s", seed, repeat, diff))
			}
			if result.TimedOut || result.Error != "" {
				return fail("java-run", fmt.Sprintf("seed %d repeat %d: timeout=%v error=%s", seed, repeat, result.TimedOut, result.Error))
			}
			if baseline == nil {
				copy := result
				baseline = &copy
			} else if diff := Compare(*baseline, result); diff != "" {
				return fail("java-nondeterminism", fmt.Sprintf("seed %d: %s", seed, diff))
			}
		}
	}
	for target, b := range p.resources {
		if e = write(filepath.Join(closureClasses, target), b); e != nil {
			return fail("resources", e.Error())
		}
	}
	for _, observation := range report.Observations {
		dir := filepath.Join(runDir, "runs", fmt.Sprintf("seed-%d-repeat-%d", observation.Seed, observation.Repeat), "java-source")
		if e = seedResources(dir, p.resources); e != nil {
			return fail("resources", e.Error())
		}
		args := append([]string{java, "-Dfile.encoding=UTF-8", "-Duser.language=en", "-Duser.country=US", "-Duser.timezone=UTC", "-cp", closureClasses, m.MainClass}, seedArgs(m.Args, observation.Seed)...)
		sourceResult := execute(ctx, c.RunTimeout, dir, args)
		if e = saveExecution(filepath.Join(dir, "..", "java-source"), sourceResult); e != nil {
			return fail("artifacts", e.Error())
		}
		sourceResult.Files, e = outputFiles(dir, m.OutputFiles, p.resources)
		if e != nil {
			return fail("java-source-output", e.Error())
		}
		if e = writeJSON(filepath.Join(dir, "..", fmt.Sprintf("java-source-%d.json", observation.Repeat)), sourceResult); e != nil {
			return fail("freeze", e.Error())
		}
		if diff := Compare(observation.Java, sourceResult); diff != "" {
			return fail("java-source-binary-parity", fmt.Sprintf("seed %d repeat %d: %s", observation.Seed, observation.Repeat, diff))
		}
	}
	if e = writeJSON(filepath.Join(runDir, "frozen/java-observations.json"), report.Observations); e != nil {
		return fail("freeze", e.Error())
	}
	transpiler := c.Transpiler
	if transpiler == "" {
		transpiler = filepath.Join(runDir, "bin/java2go")
		if !stage("transpiler-build", c.Repository, []string{"go", "build", "-o", transpiler, "./cmd/java2go"}) {
			return fail("transpiler-build", stageDetail("transpiler-build"))
		}
	}
	transpilerBytes, e := os.ReadFile(transpiler)
	if e != nil {
		return fail("transpiler-build", e.Error())
	}
	report.TranspilerSHA256 = hash(transpilerBytes)
	generated := filepath.Join(runDir, "generated")
	args = []string{transpiler, "-strict", "-maven", p.project, "-main-class", m.MainClass, "-runtime", c.Repository, "-module", "campaign.generated/" + m.Name, "-output", generated}
	for _, mapping := range p.mappings {
		args = append(args, "-dependency-source", mapping)
	}
	if !stage("transpile", runDir, args) {
		return fail("transpile", stageDetail("transpile"))
	}
	e = filepath.WalkDir(generated, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			b, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			if strings.Contains(string(b), "UNSUPPORTED:") {
				return fmt.Errorf("unsupported marker in generated source %s", path)
			}
		}
		return nil
	})
	if e != nil {
		return fail("transpile-unsupported", e.Error())
	}
	goBuild := []string{"go", "build", "-mod=mod"}
	if c.Race {
		goBuild = append(goBuild, "-race")
	}
	if !stage("go-build-all", generated, append(append([]string{}, goBuild...), "./...")) {
		return fail("go-build-all", stageDetail("go-build-all"))
	}
	binary := filepath.Join(runDir, "bin/application")
	if !stage("go-build-main", generated, append(append([]string{}, goBuild...), "-o", binary, "./cmd/app")) {
		return fail("go-build-main", stageDetail("go-build-main"))
	}
	// Stress executions cycle through the fixed inputs and compare with the
	// already repeated JVM oracle. They remain separate from its nine pairs.
	baselines := map[int]Execution{}
	observations := make([]*Observation, 0, len(report.Observations)+c.StressRuns)
	for i := range report.Observations {
		observation := &report.Observations[i]
		baselines[observation.Seed] = observation.Java
		observations = append(observations, observation)
	}
	report.StressObservations = make([]Observation, c.StressRuns)
	for i := range report.StressObservations {
		seed := m.Seeds[i%len(m.Seeds)]
		observation := &report.StressObservations[i]
		*observation = Observation{Seed: seed, Repeat: m.Repeats + 1 + i/len(m.Seeds), Java: baselines[seed]}
		observations = append(observations, observation)
	}
	for _, observation := range observations {
		dir := filepath.Join(runDir, "runs", fmt.Sprintf("seed-%d-repeat-%d", observation.Seed, observation.Repeat), "go")
		if e = seedResources(dir, p.resources); e != nil {
			return fail("resources", e.Error())
		}
		result := execute(ctx, c.RunTimeout, dir, append([]string{binary}, seedArgs(m.Args, observation.Seed)...))
		if e = saveExecution(filepath.Join(dir, "..", "go"), result); e != nil {
			return fail("artifacts", e.Error())
		}
		result.Files, e = outputFiles(dir, m.OutputFiles, p.resources)
		observation.Go = &result
		if e != nil {
			return fail("go-output-files", e.Error())
		}
		if result.TimedOut || result.Error != "" {
			return fail("go-run", fmt.Sprintf("seed %d repeat %d: timeout=%v error=%s", observation.Seed, observation.Repeat, result.TimedOut, result.Error))
		}
		if diff := Compare(observation.Java, result); diff != "" {
			return fail("parity", fmt.Sprintf("seed %d repeat %d: %s", observation.Seed, observation.Repeat, diff))
		}
	}
	report.Passed = true
	return report, nil
}
func seedArgs(args []string, seed int) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		out[i] = strings.ReplaceAll(arg, "{seed}", strconv.Itoa(seed))
	}
	return out
}
func seedResources(root string, resources map[string][]byte) error {
	if e := os.MkdirAll(root, 0755); e != nil {
		return e
	}
	for name, b := range resources {
		if e := write(filepath.Join(root, name), b); e != nil {
			return e
		}
	}
	return nil
}
func outputFiles(root string, required []string, resources map[string][]byte) (map[string]string, error) {
	files := map[string]string{}
	e := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("output symlink is unsupported: %s", path)
		}
		name, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		if initial, ok := resources[name]; ok && string(initial) == string(b) {
			return nil
		}
		files[filepath.ToSlash(name)] = hash(b)
		return nil
	})
	if e != nil {
		return nil, e
	}
	for name := range resources {
		if _, e := os.Stat(filepath.Join(root, name)); os.IsNotExist(e) {
			files[filepath.ToSlash(name)] = ""
		} else if e != nil {
			return nil, e
		}
	}
	for _, name := range required {
		b, e := os.ReadFile(filepath.Join(root, name))
		if e != nil {
			return nil, fmt.Errorf("required output %s: %w", name, e)
		}
		files[filepath.ToSlash(name)] = hash(b)
	}
	return files, nil
}
func sourceFiles(root string) ([]string, error) {
	var files []string
	e := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("source/resource symlink prohibited: %s", path)
		}
		if !d.IsDir() && strings.HasSuffix(path, ".java") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, e
}
func pom(group, id, version string, deps []Artifact) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "<project><modelVersion>4.0.0</modelVersion><groupId>%s</groupId><artifactId>%s</artifactId><version>%s</version><dependencies>", group, id, version)
	for _, d := range deps {
		fmt.Fprintf(&b, "<dependency><groupId>%s</groupId><artifactId>%s</artifactId><version>%s</version></dependency>", d.Group, d.ID, d.Version)
	}
	b.WriteString("</dependencies></project>\n")
	return []byte(b.String())
}
func prepare(c Config, m Manifest, lock Lock, runDir string, report *Report) (p prepared, err error) {
	p.resources = map[string][]byte{}
	p.project = filepath.Join(runDir, "frozen/project")
	originalPOM, e := os.ReadFile(filepath.Join(c.Fixture, m.POM))
	if e != nil {
		return p, e
	}
	report.SourceHashes["fixture/"+m.POM] = hash(originalPOM)
	if e = write(filepath.Join(runDir, "frozen/original-pom.xml"), originalPOM); e != nil {
		return p, e
	}
	if e = validatePOM(originalPOM, m, lock); e != nil {
		return p, e
	}
	for _, root := range m.SourceRoots {
		sourceRoot := filepath.Join(c.Fixture, root)
		sources, e := sourceFiles(sourceRoot)
		if e != nil {
			return p, e
		}
		if len(sources) == 0 {
			return p, fmt.Errorf("source root %s contains no Java files", root)
		}
		for _, source := range sources {
			rel, e := filepath.Rel(sourceRoot, source)
			if e != nil {
				return p, e
			}
			b, e := os.ReadFile(source)
			if e != nil {
				return p, e
			}
			target := filepath.Join(p.project, "src/main/java", rel)
			if _, e = os.Stat(target); e == nil {
				return p, fmt.Errorf("duplicate source path %s", rel)
			}
			if e = write(target, b); e != nil {
				return p, e
			}
			report.SourceHashes["fixture/"+filepath.ToSlash(filepath.Join(root, rel))] = hash(b)
			p.appSources = append(p.appSources, target)
		}
	}
	p.allSources = append(p.allSources, p.appSources...)
	var deps []Artifact
	for _, id := range m.Dependencies {
		binary, e := lock.Artifact(id, "binary")
		if e != nil {
			return p, e
		}
		source, e := lock.Artifact(id, "sources")
		if e != nil {
			return p, e
		}
		deps = append(deps, binary)
		// Copy locked binaries into frozen evidence; later cache changes cannot alter the oracle.
		binaryBytes, e := os.ReadFile(filepath.Join(c.Repository, ".campaign/cache", binary.File))
		if e != nil {
			return p, e
		}
		binaryPath := filepath.Join(runDir, "frozen/dependencies", binary.File)
		if e = write(binaryPath, binaryBytes); e != nil {
			return p, e
		}
		p.classpath = append(p.classpath, binaryPath)
		projectRoot := filepath.Join(runDir, "frozen/dependency-projects", id)
		p.mappings = append(p.mappings, binary.Group+":"+id+"="+projectRoot)
		reader, e := zip.OpenReader(filepath.Join(c.Repository, ".campaign/cache", source.File))
		if e != nil {
			return p, e
		}
		selected := map[string]bool{}
		all := false
		for _, path := range m.DependencySources[id] {
			if path == "**" {
				all = true
			} else if selected[path] {
				return p, errors.Join(fmt.Errorf("duplicate dependency source %s", path), reader.Close())
			} else {
				selected[path] = true
			}
		}
		found := map[string]bool{}
		for _, file := range reader.File {
			if file.FileInfo().IsDir() {
				continue
			}
			isJava := strings.HasSuffix(file.Name, ".java")
			if (!isJava || (!all && !selected[file.Name])) && !strings.Contains(strings.ToUpper(file.Name), "LICENSE") && !strings.Contains(strings.ToUpper(file.Name), "NOTICE") {
				continue
			}
			if !safeRelative(file.Name) {
				return p, errors.Join(fmt.Errorf("unsafe archive member %q", file.Name), reader.Close())
			}
			stream, e := file.Open()
			if e != nil {
				return p, errors.Join(e, reader.Close())
			}
			b, e := io.ReadAll(stream)
			e = errors.Join(e, stream.Close())
			if e != nil {
				return p, errors.Join(e, reader.Close())
			}
			if isJava {
				target := filepath.Join(projectRoot, "src/main/java", file.Name)
				if e = write(target, b); e != nil {
					return p, errors.Join(e, reader.Close())
				}
				p.allSources = append(p.allSources, target)
				report.SourceHashes[id+"/"+file.Name] = hash(b)
				found[file.Name] = true
			} else if e = write(filepath.Join(projectRoot, "upstream-metadata", file.Name), b); e != nil {
				return p, errors.Join(e, reader.Close())
			}
		}
		if e = reader.Close(); e != nil {
			return p, e
		}
		for path := range selected {
			if !found[path] {
				return p, fmt.Errorf("implementation source not in locked %s archive: %s", id, path)
			}
		}
		if len(found) == 0 {
			return p, fmt.Errorf("empty implementation source closure for %s", id)
		}
		if e = write(filepath.Join(projectRoot, "pom.xml"), pom(binary.Group, id, binary.Version, nil)); e != nil {
			return p, e
		}
	}
	if e = write(filepath.Join(p.project, "pom.xml"), pom("campaign.fixture", m.Name, "1", deps)); e != nil {
		return p, e
	}
	for _, resource := range m.Resources {
		path := filepath.Join(c.Fixture, resource.Source)
		info, e := os.Lstat(path)
		if e != nil {
			return p, e
		}
		add := func(source, target string) error {
			if _, ok := p.resources[target]; ok {
				return fmt.Errorf("duplicate resource target %s", target)
			}
			b, e := os.ReadFile(source)
			if e != nil {
				return e
			}
			p.resources[target] = b
			report.SourceHashes["resource/"+filepath.ToSlash(target)] = hash(b)
			return write(filepath.Join(p.project, "src/main/resources", target), b)
		}
		if info.IsDir() {
			e = filepath.WalkDir(path, func(source string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if d.Type()&os.ModeSymlink != 0 {
					return fmt.Errorf("symlink resource %s", source)
				}
				if d.IsDir() {
					return nil
				}
				rel, e := filepath.Rel(path, source)
				if e != nil {
					return e
				}
				return add(source, filepath.Join(resource.Target, rel))
			})
		} else if info.Mode()&os.ModeSymlink != 0 {
			e = fmt.Errorf("symlink resource %s", path)
		} else {
			e = add(path, resource.Target)
		}
		if e != nil {
			return p, e
		}
	}
	sort.Strings(p.appSources)
	sort.Strings(p.allSources)
	return p, nil
}
func validatePOM(data []byte, m Manifest, lock Lock) error {
	var model struct {
		Dependencies []struct {
			Group   string `xml:"groupId"`
			ID      string `xml:"artifactId"`
			Version string `xml:"version"`
			Scope   string `xml:"scope"`
		} `xml:"dependencies>dependency"`
	}
	if e := xml.Unmarshal(data, &model); e != nil {
		return e
	}
	want := map[string]Artifact{}
	for _, id := range m.Dependencies {
		a, e := lock.Artifact(id, "binary")
		if e != nil {
			return e
		}
		want[a.Group+":"+a.ID] = a
	}
	found := map[string]bool{}
	for _, d := range model.Dependencies {
		if d.Scope == "test" {
			continue
		}
		key := d.Group + ":" + d.ID
		a, ok := want[key]
		if !ok {
			return fmt.Errorf("POM dependency %s has no declared locked implementation closure", key)
		}
		if a.Version != d.Version {
			return fmt.Errorf("POM dependency %s version %s differs from lock %s", key, d.Version, a.Version)
		}
		found[key] = true
	}
	for key := range want {
		if !found[key] {
			return fmt.Errorf("manifest dependency %s is absent from fixture POM", key)
		}
	}
	return nil
}
