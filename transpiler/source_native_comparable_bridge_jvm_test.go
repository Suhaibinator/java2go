package transpiler

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// Each complete project and observation was frozen before the bridge producer.
// Repeating the same full observation checks JVM and generated race parity.
func comparableBridge136FullProject(t *testing.T, name string) {
	t.Helper()
	data, err := os.ReadFile("testdata/comparablebridge136/observations.json")
	if err != nil {
		t.Fatal(err)
	}
	var oracles map[string]struct {
		Stdout string `json:"stdout"`
		Stderr string `json:"stderr"`
	}
	if err = json.Unmarshal(data, &oracles); err != nil {
		t.Fatal(err)
	}
	oracle, ok := oracles[name]
	if !ok || oracle.Stderr != "" {
		t.Fatal("missing complete frozen JDK oracle")
	}
	root := filepath.Join("testdata/comparablebridge136", name)
	files := map[string]string{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(body)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	observations := []campaignCompilerProjectObservation{
		{name: "first", stdout: oracle.Stdout, stderr: oracle.Stderr},
		{name: "second", stdout: oracle.Stdout, stderr: oracle.Stderr},
		{name: "third", stdout: oracle.Stdout, stderr: oracle.Stderr},
	}
	runCampaignCompilerStrictProjectObservations(t, files, "probe.Main", observations)
}

func TestComparableBridge136DeclaredBoundErasureStrictJVM(t *testing.T) {
	comparableBridge136FullProject(t, "Erasure")
}
func TestComparableBridge136VirtualSelectionStrictJVM(t *testing.T) {
	comparableBridge136FullProject(t, "Selection")
}
func TestComparableBridge136OverloadCollisionFirstOrderStrictJVM(t *testing.T) {
	comparableBridge136FullProject(t, "CollisionA")
}
func TestComparableBridge136OverloadCollisionReverseOrderStrictJVM(t *testing.T) {
	comparableBridge136FullProject(t, "CollisionB")
}
func TestComparableBridge136DeclarationNamespaceStrictJVM(t *testing.T) {
	comparableBridge136FullProject(t, "Namespace")
}
