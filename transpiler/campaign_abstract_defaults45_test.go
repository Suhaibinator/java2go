package transpiler

import ("os";"testing")

func TestCampaignAbstractCollectionDefaultsJVM45(t *testing.T) {
 source, err := os.ReadFile("testdata/abstract_defaults45/abstract-collection.java"); if err != nil {t.Fatal(err)}
 expected, err := os.ReadFile("testdata/abstract_defaults45/abstract-collection.stdout"); if err != nil {t.Fatal(err)}
 files:=map[string]string{"pom.xml": `<project><groupId>probe</groupId><artifactId>abstract-defaults</artifactId><version>1</version></project>`,"src/main/java/probe/collection/Main.java":string(source)}
 runCampaignCompilerStrictProjectOracle(t,files,"probe.collection.Main",string(expected))
}

func TestCampaignAbstractSetDefaultsJVM45(t *testing.T) {
 source, err := os.ReadFile("testdata/abstract_defaults45/abstract-set.java"); if err != nil {t.Fatal(err)}
 expected, err := os.ReadFile("testdata/abstract_defaults45/abstract-set.stdout"); if err != nil {t.Fatal(err)}
 files:=map[string]string{"pom.xml": `<project><groupId>probe</groupId><artifactId>abstract-defaults</artifactId><version>1</version></project>`,"src/main/java/probe/set/Main.java":string(source)}
 runCampaignCompilerStrictProjectOracle(t,files,"probe.set.Main",string(expected))
}

