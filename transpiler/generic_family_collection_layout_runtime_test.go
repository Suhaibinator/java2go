package transpiler

import (
	"os"
	"testing"
)

func TestGenericFamilyCollectionCarrierJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/generic_collection_carrier/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>collection-carrier</artifactId><version>1</version></project>`,
		"src/main/java/genericcollectioncarrier/Main.java": string(source),
	}, "genericcollectioncarrier.Main", "ok|true|true|true|true\n")
}

func TestGenericFamilyCollectionCarrierNominalJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/generic_collection_carrier_nominal/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>collection-carrier-nominal</artifactId><version>1</version></project>`,
		"src/main/java/genericcollectioncarriernominal/Main.java": string(source),
	}, "genericcollectioncarriernominal.Main", "ok|true|true|true|true|true\n")
}
