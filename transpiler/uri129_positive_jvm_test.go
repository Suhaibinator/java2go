package transpiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestURI129AdmittedProvenanceStrictJVM(t *testing.T) {
	root := "testdata/uri129_positive"
	files := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if filepath.Ext(path) != ".java" && filepath.Base(path) != "pom.xml" {
			return nil
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(name)] = string(raw)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "observation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var row struct {
		Stdout, Stderr string
		Exit           int `json:"exit_code"`
	}
	if err = json.Unmarshal(raw, &row); err != nil {
		t.Fatal(err)
	}
	if row.Exit != 0 {
		t.Fatalf("JDK oracle failed: %+v", row)
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", []campaignCompilerProjectObservation{{name: "all", stdout: row.Stdout, stderr: row.Stderr}})
}
