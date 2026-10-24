package transpiler

import (
	"os"
	"testing"
)

func TestCollectionsEmptySetStrictJDK21(t *testing.T) {
	cases := []struct{ name, output string }{
		{"EmptySetControl", "identity=true\nnominal=true\nmutation=true\niteration=true\ncallbacks=true\n"},
		{"EmptySetBinderControl", "captured=true\nnominal-array-binder=true\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			source, err := os.ReadFile("testdata/collections_empty_set_v1/" + c.name + ".java")
			if err != nil {
				t.Fatal(err)
			}
			observations := []campaignCompilerProjectObservation{{name: "repeat1", stdout: c.output}, {name: "repeat2", stdout: c.output}, {name: "repeat3", stdout: c.output}}
			runCampaignCompilerStrictProjectObservations(t, map[string]string{"pom.xml": `<project><groupId>emptyset</groupId><artifactId>factory</artifactId><version>1</version></project>`, "src/main/java/emptyset/" + c.name + ".java": string(source)}, "emptyset."+c.name, observations)
		})
	}
}
