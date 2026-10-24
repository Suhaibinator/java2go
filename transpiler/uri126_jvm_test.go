package transpiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestURI126StrictMultiPackageJVM(t *testing.T) {
	root := filepath.Join("testdata", "uri126_core")
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
	raw, err := os.ReadFile(filepath.Join(root, "observations.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Args   []string `json:"args"`
		Stdout string   `json:"stdout"`
		Stderr string   `json:"stderr"`
		Exit   int      `json:"exit_code"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Fatalf("need all five original JDK observations; got %d", len(rows))
	}
	observations := make([]campaignCompilerProjectObservation, len(rows))
	for i, row := range rows {
		if row.Exit != 0 || len(row.Args) != 1 {
			t.Fatalf("invalid oracle metadata %+v", row)
		}
		observations[i] = campaignCompilerProjectObservation{name: row.Args[0], args: row.Args, stdout: row.Stdout, stderr: row.Stderr}
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", observations)
}
