package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func reflectTypeObjectRoutingProject(t *testing.T, name, main string) {
	t.Helper()
	root := filepath.Join("testdata", "reflect_type_equals131")
	source, err := os.ReadFile(filepath.Join(root, name+".java"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, name+".expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"pom.xml":                         `<project><modelVersion>4.0.0</modelVersion><groupId>reflection</groupId><artifactId>object-routing</artifactId><version>1</version></project>`,
		"src/main/java/" + name + ".java": string(source),
	}
	if name == "SourceGuardsOracle" {
		foreign, err := os.ReadFile(filepath.Join(root, "foreign", "Type.java"))
		if err != nil {
			t.Fatal(err)
		}
		files["src/main/java/foreign/Type.java"] = string(foreign)
	}
	runCampaignCompilerStrictProjectOracle(t, files, main, string(want))
}

func TestReflectTypeObjectRoutingJDK21(t *testing.T) {
	reflectTypeObjectRoutingProject(t, "RoutingOracle", "eqroute.RoutingOracle")
}
func TestReflectTypeObjectSelectedOverloadJDK21(t *testing.T) {
	reflectTypeObjectRoutingProject(t, "OverloadOracle", "eqoverload.OverloadOracle")
}
func TestReflectTypeObjectMethodReferenceJDK21(t *testing.T) {
	reflectTypeObjectRoutingProject(t, "MethodReferenceOracle", "eqmethodref.MethodReferenceOracle")
}

func TestReflectTypeObjectSourceGuardsJDK21(t *testing.T) {
	reflectTypeObjectRoutingProject(t, "SourceGuardsOracle", "eqguards.SourceGuardsOracle")
}
