package transpiler

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnumValuesTextJDK21(t *testing.T) {
	for _, item := range []struct{ name, main string }{
		{"arrays", "enumfrontier.ArraysControl"}, {"text", "enumfrontier.TextControl"},
		{"init", "enumfrontier.InitControl"}, {"guards", "enumfrontier.GuardsControl"},
		{"namespace", "enumconsumer.NamespaceControl"}, {"messages", "enumfrontier.MessagesControl"},
		{"collisions", "enumfrontier.CollisionsControl"},
		{"constructor", "enumfrontier.ConstructorControl"},
	} {
		t.Run(item.name, func(t *testing.T) {
			root := filepath.Join("testdata", "enum_values_text_v1", item.name)
			files := map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>enumfrontier</groupId><artifactId>enum-frontier</artifactId><version>1</version></project>`}
			err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() || !strings.HasSuffix(path, ".java") {
					return nil
				}
				content, e := os.ReadFile(path)
				if e != nil {
					return e
				}
				rel, e := filepath.Rel(root, path)
				if e != nil {
					return e
				}
				files["src/main/java/"+filepath.ToSlash(rel)] = string(content)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			expected, err := os.ReadFile(filepath.Join(root, "expected.stdout"))
			if err != nil {
				t.Fatal(err)
			}
			runCampaignCompilerStrictProjectOracle(t, files, item.main, string(expected))
		})
	}
}
