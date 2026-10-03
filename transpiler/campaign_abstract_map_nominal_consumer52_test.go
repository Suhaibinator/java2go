package transpiler

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestCampaignAbstractMapNominalConsumerJVM52(t *testing.T) {
	fixture := filepath.Join("testdata", "abstract_map_gap47", "nominal-consumer")
	files := map[string]string{}
	project := filepath.Join(fixture, "project")
	err := filepath.WalkDir(project, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		name, err := filepath.Rel(project, path)
		if err != nil {
			return err
		}
		if name != "pom.xml" && filepath.Ext(name) != ".java" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(name)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, seed := range []int{17, 41, 97} {
		t.Run(strconv.Itoa(seed), func(t *testing.T) {
			expected, err := os.ReadFile(filepath.Join(fixture, "expected", "seed-"+strconv.Itoa(seed)+".stdout"))
			if err != nil {
				t.Fatal(err)
			}
			expectedError, err := os.ReadFile(filepath.Join(fixture, "expected", "seed-"+strconv.Itoa(seed)+".stderr"))
			if err != nil {
				t.Fatal(err)
			}
			if len(expectedError) != 0 {
				t.Fatalf("Independent JVM stderr is not empty: %q", expectedError)
			}
			runCampaignCompilerStrictProjectOracle47Args(t, files, "probe.app.Main", string(expected), strconv.Itoa(seed))
		})
	}
}
