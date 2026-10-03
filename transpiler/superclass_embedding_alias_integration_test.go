package transpiler

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// These independent source projects exercise superclass storage selectors while
// retaining Java virtual dispatch, reference identity, and exact generic views.
// Each case obtains a fresh Java 21 oracle before strict generated Go execution.
func TestSuperclassEmbeddingAliasProjectJVM(t *testing.T) {
	cases := []struct{ name, expected string }{
		{"inherited", "11\n"},
		{"override_super", "17:11:17\n"},
		{"constructor_callback", "9:7:11:7:4:11\n"},
		{"generic_views", "11:seed:echo:true:true:true:true\n"},
		{"local_anonymous", "11:11:17:11\n"},
		{"nested_qualified", "11:11:true:5\n"},
		{"dollar", "13:13\n"},
		{"alias_namespace", "28:11:7:true\n"},
		{"field_identity", "4:6:11:4:6\n"},
		{"repeated_local", "11:17\n"},
		{"constrained_swapped", "11:7:seed:true\n"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			directory := filepath.Join("testdata", "superclass_embedding_alias", test.name)
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
			runCampaignCompilerStrictProjectOracle(t, files, "app.Main", test.expected)
		})
	}
}
