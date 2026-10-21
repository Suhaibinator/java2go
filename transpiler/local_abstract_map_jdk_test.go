package transpiler

import (
	"os"
	"testing"
)

func TestLocalAbstractMapStrictJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/local_abstract_map_v1/LocalAbstractMapControl.java")
	if err != nil {
		t.Fatal(err)
	}
	observations := []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "native-local=true\nsource-shadow=true\n"},
		{name: "repeat2", stdout: "native-local=true\nsource-shadow=true\n"},
		{name: "repeat3", stdout: "native-local=true\nsource-shadow=true\n"},
	}
	runCampaignCompilerStrictProjectObservations(t, map[string]string{"pom.xml": `<project><groupId>local</groupId><artifactId>map</artifactId><version>1</version></project>`, "src/main/java/localmap/LocalAbstractMapControl.java": string(source)}, "localmap.LocalAbstractMapControl", observations)
}
