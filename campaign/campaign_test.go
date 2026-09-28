package campaign

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestRejectsSkippedCoverage(t *testing.T) {
	good := Manifest{Name: "sample", MainClass: "app.Main", SourceRoots: []string{"src"}, POM: "pom.xml", Seeds: []int{17, 41, 97}, Repeats: 3}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Manifest){
		func(m *Manifest) { m.Repeats = 1 },
		func(m *Manifest) { m.Seeds = []int{17} },
		func(m *Manifest) { m.Dependencies = []string{"gson"} },
		func(m *Manifest) { m.OutputFiles = []string{"../escape"} },
	} {
		m := good
		mutate(&m)
		if m.Validate() == nil {
			t.Fatal("accepted incomplete/unsafe manifest")
		}
	}
}
func TestParityIncludesExitStderrAndFiles(t *testing.T) {
	base := Execution{ExitCode: 0, Stdout: "ok\n", Files: map[string]string{"output": "abc"}}
	variants := []Execution{
		{ExitCode: 1, Stdout: "ok\n", Files: base.Files},
		{Stdout: "ok\n", Stderr: "warning", Files: base.Files},
		{Stdout: "ok\n", Files: map[string]string{}},
		{Stdout: "ok\n", Files: base.Files, TimedOut: true},
		{Stdout: "ok\n", Files: base.Files, Error: "start failed"},
	}
	if Compare(base, base) != "" {
		t.Fatal("identical observations differed")
	}
	for _, got := range variants {
		if Compare(base, got) == "" {
			t.Fatalf("false parity: %+v", got)
		}
	}
}
func TestLockRejectsModifiedArtifact(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".campaign/cache"), 0755)
	path := filepath.Join(root, ".campaign/cache/lib.jar")
	os.WriteFile(path, []byte("modified"), 0644)
	lock := Lock{SchemaVersion: 1, Artifacts: []Artifact{{ID: "lib", File: "lib.jar", SHA256: strings.Repeat("0", 64)}}}
	if err := lock.Verify(root); err == nil {
		t.Fatal("accepted changed artifact")
	}
}
func TestLoadRejectsUnknownManifestFields(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "fixture.json"), []byte(`{"status":"known-gap"}`), 0644)
	if _, err := LoadManifest(dir); err == nil {
		t.Fatal("accepted passing-by-declaration field")
	}
}
