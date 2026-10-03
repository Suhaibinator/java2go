package transpiler

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// UTF-8 hex identities need a delimiter before a collision suffix: Derived's
// forty-first numeric suffix must not equal DerivedA's unsuffixed hex identity.
// Legal Java methods occupy the first forty-one candidate selector spellings.
func TestSuperclassEmbeddingAliasInjectivityProjectJVM(t *testing.T) {
	directory := filepath.Join("testdata", "superclass_embedding_alias_injectivity")
	files := make(map[string]string)
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Base(path) == "oracle.json" {
			return nil
		}
		relative, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(contents)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, files, "app.Main", "11:11:0:40\n")
}
