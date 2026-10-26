package transpiler

import (
	"os"
	"testing"
)

func TestGenericFamilySourceClassStorageDateJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/date_source_class_storage/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>date-class-storage</artifactId><version>1</version></project>`, "src/main/java/sourceclassstorageobject/Main.java": string(source)}, "sourceclassstorageobject.Main", "true|true|5678|678000000|true|true|true|true\n", sourceInterfaceStorageParentCLI(t)...)
}

func TestGenericFamilySourceClassStorageNestedAnonymousJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/date_source_class_storage_nested/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>date-nested-storage</artifactId><version>1</version></project>`, "src/main/java/sourceclassstoragenested/Main.java": string(source)}, "sourceclassstoragenested.Main", "true|true|5678|678000000|true|true|true|true\n", sourceInterfaceStorageParentCLI(t)...)
}
