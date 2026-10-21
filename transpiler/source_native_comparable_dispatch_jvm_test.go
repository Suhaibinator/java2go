package transpiler

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestComparableDispatch136ProtocolStrictJVM(t *testing.T) {
	data, err := os.ReadFile("testdata/comparabledispatch136/observations.json")
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
	oracle, ok := observations["Protocol"]
	if !ok || oracle.Stderr != "" {
		t.Fatal("missing whole frozen JDK oracle")
	}
	root := "testdata/comparabledispatch136/Protocol"
	files := map[string]string{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
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
	runs := []campaignCompilerProjectObservation{{name: "first", stdout: oracle.Stdout, stderr: oracle.Stderr}, {name: "second", stdout: oracle.Stdout, stderr: oracle.Stderr}, {name: "third", stdout: oracle.Stdout, stderr: oracle.Stderr}}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", runs)
}
