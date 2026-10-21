package transpiler

import (
	"os"
	"testing"
)

func TestMapCapturedHeaderStrictJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/map_captured_header/CaptureMapControl.java")
	if err != nil {
		t.Fatal(err)
	}
	observations := []campaignCompilerProjectObservation{{name: "repeat1", stdout: "true:true:true:true\n"}, {name: "repeat2", stdout: "true:true:true:true\n"}, {name: "repeat3", stdout: "true:true:true:true\n"}}
	runCampaignCompilerStrictProjectObservations(t, map[string]string{"pom.xml": `<project><groupId>capture</groupId><artifactId>map</artifactId><version>1</version></project>`, "src/main/java/staticabi/CaptureMapControl.java": string(source)}, "staticabi.CaptureMapControl", observations)
}

// Independent control isolates source Function transport from local capture
// binding and native AbstractMap/Collections frontiers. Full capture remains.
func TestMapSourceFunctionProtocolStrictJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/map_captured_header/SourceFunctionControl.java")
	if err != nil {
		t.Fatal(err)
	}
	observations := []campaignCompilerProjectObservation{{name: "repeat1", stdout: "true:true:true\n"}, {name: "repeat2", stdout: "true:true:true\n"}, {name: "repeat3", stdout: "true:true:true\n"}}
	runCampaignCompilerStrictProjectObservations(t, map[string]string{"pom.xml": `<project><groupId>callbackabi</groupId><artifactId>map</artifactId><version>1</version></project>`, "src/main/java/callbackabi/SourceFunctionControl.java": string(source)}, "callbackabi.SourceFunctionControl", observations)
}
