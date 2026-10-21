package transpiler

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func interface136FullProject(t *testing.T, name string) {
	t.Helper()
	data, err := os.ReadFile("testdata/interface136/observations.json")
	if err != nil {
		t.Fatal(err)
	}
	var observations map[string]struct {
		Stdout string `json:"stdout"`
		Stderr string `json:"stderr"`
	}
	if err = json.Unmarshal(data, &observations); err != nil {
		t.Fatal(err)
	}
	oracle, ok := observations[name]
	if !ok || oracle.Stderr != "" {
		t.Fatal("missing complete frozen JDK oracle")
	}
	root := filepath.Join("testdata/interface136", name)
	files := map[string]string{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		relative, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		body, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		files[filepath.ToSlash(relative)] = string(body)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	entry := "cachecontrol." + name
	if name == "Nominal" {
		entry = "app.Main"
	}
	runCampaignCompilerStrictProjectOracle(t, files, entry, oracle.Stdout)
}

func TestInterface136BridgeOriginalStrictJVM(t *testing.T) { interface136FullProject(t, "Bridge") }
func TestInterface136ExecutionOriginalStrictJVM(t *testing.T) {
	interface136FullProject(t, "Execution")
}
func TestInterface136NominalHierarchyStrictJVM(t *testing.T) { interface136FullProject(t, "Nominal") }
