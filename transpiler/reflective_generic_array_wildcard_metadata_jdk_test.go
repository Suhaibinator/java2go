package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func reflectiveArrayWildcardProject(t *testing.T, name, main string, additional ...string) {
	t.Helper()
	root := filepath.Join("testdata", "reflective_generic_array_wildcard_metadata")
	files := map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>metadata</groupId><artifactId>arrays-wildcards</artifactId><version>1</version></project>`}
	for _, filename := range append([]string{name + ".java"}, additional...) {
		source, err := os.ReadFile(filepath.Join(root, filename))
		if err != nil {
			t.Fatal(err)
		}
		files[filepath.Join("src/main/java", filename)] = string(source)
	}
	want, err := os.ReadFile(filepath.Join(root, name+".expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, files, main, string(want))
}
func TestReflectiveGenericArrayWildcardWitnessJDK21(t *testing.T) {
	reflectiveArrayWildcardProject(t, "Witness", "metadatawitness.Witness")
}

func TestReflectiveGenericArrayWildcardMatrixJDK21(t *testing.T) {
	reflectiveArrayWildcardProject(t, "Matrix", "metadatamatrix.Matrix", "foreign/Class.java")
}
func TestReflectiveGenericMetadataObjectDefaultsJDK21(t *testing.T) {
	reflectiveArrayWildcardProject(t, "Defaults", "metadatadefaults.Defaults")
}

func TestReflectiveGenericMetadataWildcardObjectDefaultsJDK21(t *testing.T) {
	reflectiveArrayWildcardProject(t, "DefaultWildcards", "metadatadefaults.DefaultWildcards", "Defaults.java")
}
