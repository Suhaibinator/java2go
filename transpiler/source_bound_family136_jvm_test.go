package transpiler

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func sourceBound136Project(t *testing.T, name, main string) {
	t.Helper()
	root := filepath.Join("testdata/source_bound136", name)
	expected, err := os.ReadFile(filepath.Join(root, "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "expected.stdout" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err == nil {
			files[filepath.ToSlash(rel)] = string(b)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectObservations(t, files, main, []campaignCompilerProjectObservation{{name: "first", stdout: string(expected)}, {name: "second", stdout: string(expected)}, {name: "third", stdout: string(expected)}})
}
func TestSourceBound136StorageConstructorLateViewsJDK21(t *testing.T) {
	sourceBound136Project(t, "storage", "boundproof.Main")
}
func TestSourceBound136ImportedDeclarationOriginJDK21(t *testing.T) {
	sourceBound136Project(t, "imported", "consumer.Main")
}

func TestSourceBound136EqualityNullAbruptJDK21(t *testing.T) {
	sourceBound136Project(t, "equality", "identityproof.Main")
}
