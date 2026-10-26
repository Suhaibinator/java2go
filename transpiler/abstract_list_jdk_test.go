package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAbstractListActualGsonWrapperJDK21(t *testing.T) {
	root := filepath.Join("testdata", "abstract_list_gson_wrapper")
	files := map[string]string{}
	if err := filepath.WalkDir(filepath.Join(root, "project"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		raw, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		relative, e := filepath.Rel(filepath.Join(root, "project"), path)
		files[filepath.ToSlash(relative)] = string(raw)
		return e
	}); err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(root, "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	real, err := os.ReadFile(filepath.Join("testdata", "gson_generic_parent_origin", "dependencies", "src", "com", "google", "gson", "internal", "NonNullElementWrapperList.java"))
	if err != nil {
		t.Fatal(err)
	}
	if files["src/main/java/com/google/gson/internal/NonNullElementWrapperList.java"] != string(real) {
		t.Fatal("Gson dependency wrapper changed")
	}
	runCampaignCompilerStrictProjectOracle(t, files, "probe.Main", string(expected))
}
