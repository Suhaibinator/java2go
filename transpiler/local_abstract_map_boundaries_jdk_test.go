package transpiler

import (
	"os"
	"strings"
	"testing"
)

func TestLocalAbstractMapFocusedBoundariesJDK21(t *testing.T) {
	original, err := os.ReadFile("testdata/local_abstract_map_v1/LocalAbstractMapControl.java")
	if err != nil {
		t.Fatal(err)
	}
	marker := "            return defaults && views && sourceBridge && callbacks && live && view.isEmpty() && gets > 0 && puts == 3 && entries > 0 && execution;"
	if strings.Count(string(original), marker) != 1 {
		t.Fatal("original acceptance return is not unique")
	}
	trace := `            System.out.println("boundaries=" + defaults + "," + views + "," + sourceBridge + "," + callbacks + "," + live + "," + (gets > 0) + "," + (puts == 3) + "," + (entries > 0) + "," + execution);
`
	source := strings.Replace(string(original), marker, trace+marker, 1)
	runCampaignCompilerStrictProjectObservations(t, map[string]string{"pom.xml": `<project><groupId>local</groupId><artifactId>map-boundaries</artifactId><version>1</version></project>`, "src/main/java/localmap/LocalAbstractMapControl.java": source}, "localmap.LocalAbstractMapControl", []campaignCompilerProjectObservation{{name: "boundary-trace", stdout: "boundaries=true,true,true,true,true,true,true,true,true\nnative-local=true\nsource-shadow=true\n"}})
}
