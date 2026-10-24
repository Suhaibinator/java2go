package transpiler

import (
	"os"
	"testing"
)

func TestLocalAbstractMapPollutionStrictJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/local_abstract_map_v1/LocalAbstractMapPollutionControl.java")
	if err != nil {
		t.Fatal(err)
	}
	observations := []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "raw-post-write=true\n"},
		{name: "repeat2", stdout: "raw-post-write=true\n"},
		{name: "repeat3", stdout: "raw-post-write=true\n"},
	}
	runCampaignCompilerStrictProjectObservations(t, map[string]string{"pom.xml": `<project><groupId>local</groupId><artifactId>pollution</artifactId><version>1</version></project>`, "src/main/java/localmap/LocalAbstractMapPollutionControl.java": string(source)}, "localmap.LocalAbstractMapPollutionControl", observations)
}
