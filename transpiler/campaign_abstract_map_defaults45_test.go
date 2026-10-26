package transpiler
import ("os";"testing")
func TestCampaignAbstractMapDefaultsJVM45(t *testing.T) {
 source, err := os.ReadFile("testdata/abstract_defaults45/abstract-map.java"); if err != nil { t.Fatal(err) }
 expected, err := os.ReadFile("testdata/abstract_defaults45/abstract-map.stdout"); if err != nil { t.Fatal(err) }
 files:=map[string]string{"pom.xml": `<project><groupId>probe</groupId><artifactId>abstract-map</artifactId><version>1</version></project>`, "src/main/java/probe/map/Main.java":string(source)}
 runCampaignCompilerStrictProjectOracle(t,files,"probe.map.Main",string(expected))
}
