package campaign

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecutionTimeoutNeverPasses(t *testing.T) {
	r := execute(context.Background(), 40*time.Millisecond, t.TempDir(), []string{"sh", "-c", "sleep 30"})
	if !r.TimedOut {
		t.Fatalf("timeout not classified: %+v", r)
	}
	if Compare(r, r) == "" {
		t.Fatal("identical timeouts passed parity")
	}
	if r.DurationMS > 2000 {
		t.Fatalf("timed-out process tree not killed promptly: %dms", r.DurationMS)
	}
}
func TestExecutionStartFailureNeverPasses(t *testing.T) {
	r := execute(context.Background(), time.Second, t.TempDir(), []string{filepath.Join(t.TempDir(), "missing")})
	if r.Error == "" || Compare(r, r) == "" {
		t.Fatalf("start failure passed: %+v", r)
	}
}
func TestOracleSnapshotsAreAuthority(t *testing.T) {
	root := t.TempDir()
	m := Manifest{Seeds: []int{17, 41, 97}}
	for _, seed := range []string{"17", "41", "97"} {
		if e := os.WriteFile(filepath.Join(root, "expected.seed-"+seed+".stdout"), []byte("frozen\n"), 0644); e != nil {
			t.Fatal(e)
		}
	}
	expected, e := loadExpected(root, m)
	if e != nil {
		t.Fatal(e)
	}
	if compareExpected(expected[17], Execution{Stdout: "changed\n"}) == "" {
		t.Fatal("accepted changed Java output")
	}
	if compareExpected(expected[17], Execution{Stdout: "frozen\n", ExitCode: 1}) == "" {
		t.Fatal("accepted changed Java exit code")
	}
}
func TestFixtureFingerprintDetectsOracleAndSourceChanges(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Main.java")
	if e := os.WriteFile(path, []byte("before"), 0644); e != nil {
		t.Fatal(e)
	}
	before, e := fixtureHashes(root)
	if e != nil {
		t.Fatal(e)
	}
	if err := os.WriteFile(path, []byte("after"), 0644); err != nil {
		t.Fatal(err)
	}
	after, e := fixtureHashes(root)
	if e != nil {
		t.Fatal(e)
	}
	if before["Main.java"] == after["Main.java"] {
		t.Fatal("source edit escaped fingerprint")
	}
	if err := os.WriteFile(filepath.Join(root, "expected.seed-17.stdout"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	after, e = fixtureHashes(root)
	if e != nil {
		t.Fatal(e)
	}
	if after["expected.seed-17.stdout"] == "" {
		t.Fatal("oracle not fingerprinted")
	}
}
func TestImplementationFingerprintIncludesDirtyRuntime(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = root
	if out, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("git init: %s %v", out, e)
	}
	for _, arg := range [][]string{{"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-qm", "base"}} {
		cmd = exec.Command("git", arg...)
		cmd.Dir = root
		if out, e := cmd.CombinedOutput(); e != nil {
			t.Fatalf("git commit: %s %v", out, e)
		}
	}
	for _, name := range []string{"astutil", "nodeutil", "parsing", "project", "symbol", "stdjava", "transpiler", "cmd/java2go", "campaign", "cmd/javacampaign"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"api.go", "go.mod", "go.sum", "stdjava/runtime.go"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("before"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	before, e := implementationFingerprint(root)
	if e != nil {
		t.Fatal(e)
	}
	if err := os.WriteFile(filepath.Join(root, "stdjava/runtime.go"), []byte("after"), 0644); err != nil {
		t.Fatal(err)
	}
	after, e := implementationFingerprint(root)
	if e != nil {
		t.Fatal(e)
	}
	if before.Revision != after.Revision || before.SHA256 == after.SHA256 {
		t.Fatal("dirty runtime edit not detected independently of revision")
	}
}
func TestDependencySourceComesFromLockedArchive(t *testing.T) {
	root := t.TempDir()
	fixture := filepath.Join(root, "fixture")
	run := filepath.Join(root, "run")
	source := []byte("package vendor; public class Real { public int value() { return 7; } }\n")
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	member, e := zw.Create("vendor/Real.java")
	if e != nil {
		t.Fatal(e)
	}
	if _, err := member.Write(source); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := write(filepath.Join(root, ".campaign/cache/lib-sources.jar"), archive.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := write(filepath.Join(root, ".campaign/cache/lib.jar"), []byte("binary")); err != nil {
		t.Fatal(err)
	}
	// A locally modified extracted cache is deliberately not trusted.
	if err := write(filepath.Join(root, ".campaign/sources/lib-1/vendor/Real.java"), []byte("FAKE STUB")); err != nil {
		t.Fatal(err)
	}
	if err := write(filepath.Join(fixture, "src/app/Main.java"), []byte("package app; public class Main {}")); err != nil {
		t.Fatal(err)
	}
	if err := write(filepath.Join(fixture, "pom.xml"), []byte("<project><dependencies><dependency><groupId>vendor</groupId><artifactId>lib</artifactId><version>1</version></dependency></dependencies></project>")); err != nil {
		t.Fatal(err)
	}
	m := Manifest{Name: "test", POM: "pom.xml", SourceRoots: []string{"src"}, Dependencies: []string{"lib"}, DependencySources: map[string][]string{"lib": {"vendor/Real.java"}}}
	lock := Lock{Artifacts: []Artifact{{ID: "lib", Kind: "binary", Group: "vendor", Version: "1", File: "lib.jar"}, {ID: "lib", Kind: "sources", Group: "vendor", Version: "1", File: "lib-sources.jar"}}}
	report := Report{SourceHashes: map[string]string{}}
	_, e = prepare(Config{Repository: root, Fixture: fixture}, m, lock, run, &report)
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(run, "frozen/dependency-projects/lib/src/main/java/vendor/Real.java"))
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(b, source) {
		t.Fatal("dependency source was modified or replaced")
	}
	m.DependencySources["lib"] = []string{"vendor/Missing.java"}
	_, e = prepare(Config{Repository: root, Fixture: fixture}, m, lock, filepath.Join(root, "missing-run"), &report)
	if e == nil || !strings.Contains(e.Error(), "not in locked") {
		t.Fatalf("missing closure source silently omitted: %v", e)
	}
}

func TestMavenFailureClassification(t *testing.T) {
	for _, tc := range []struct {
		result Execution
		want   string
	}{{Execution{TimedOut: true}, "maven-timeout"}, {Execution{Stderr: "Plugin could not be resolved"}, "maven-resolution"}, {Execution{Stdout: "Could not resolve dependencies"}, "maven-resolution"}, {Execution{Stderr: "compilation failure"}, "maven-build"}} {
		if got := classifyMavenFailure(tc.result); got != tc.want {
			t.Fatalf("got %s want %s", got, tc.want)
		}
	}
}
func TestOutputParityDetectsDeletedResource(t *testing.T) {
	root := t.TempDir()
	resources := map[string][]byte{"input.txt": []byte("input")}
	if e := seedResources(root, resources); e != nil {
		t.Fatal(e)
	}
	before, e := outputFiles(root, nil, resources)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(filepath.Join(root, "input.txt")); e != nil {
		t.Fatal(e)
	}
	after, e := outputFiles(root, nil, resources)
	if e != nil {
		t.Fatal(e)
	}
	if Compare(Execution{Files: before}, Execution{Files: after}) == "" {
		t.Fatal("resource deletion escaped parity")
	}
}
func TestRawStreamArtifactsPreserveInvalidUTF8(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "java")
	raw := string([]byte{0xff, 0x00, 0xfe})
	if e := saveExecution(prefix, Execution{Stdout: raw, Stderr: raw}); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(prefix + ".stdout")
	if e != nil || string(b) != raw {
		t.Fatalf("raw stdout damaged: %x %v", b, e)
	}
}
