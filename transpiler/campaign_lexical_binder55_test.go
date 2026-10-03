package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCampaignLexicalBinderCollisionJVM55(t *testing.T) {
	runLexicalBinder55Project(t, "binder-collision")
}

func TestCampaignLexicalBinderRenamedControlJVM55(t *testing.T) {
	runLexicalBinder55Project(t, "binder-renamed-control")
}

func runLexicalBinder55Project(t *testing.T, name string) {
	t.Helper()
	root := filepath.Join("testdata", "lexical_binder55", name)
	project := filepath.Join(root, "project")
	files := map[string]string{}
	if err := filepath.WalkDir(project, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		value, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(project, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(value)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	oracle, err := os.ReadFile(filepath.Join(root, "oracle.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := os.ReadFile(filepath.Join(root, "oracle.stderr"))
	if err != nil || len(stderr) != 0 {
		t.Fatalf("independent oracle stderr: %q (%v)", stderr, err)
	}
	runCampaignCompilerStrictProjectOracle(t, files, "probe.app.Main", string(oracle))
}
