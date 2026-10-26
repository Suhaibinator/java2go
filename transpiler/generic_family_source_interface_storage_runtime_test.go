package transpiler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGenericFamilySourceInterfaceStorageJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/source_interface_storage/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>source-interface-storage</artifactId><version>1</version></project>`,
		"src/main/java/sourceinterfacestorage/Main.java": string(source),
	}, "sourceinterfacestorage.Main", "ok|true|true|true|true\n", sourceInterfaceStorageParentCLI(t)...)
}

func sourceInterfaceStorageParentCLI(t *testing.T) []*campaignCompilerParentCLI {
	t.Helper()
	binary := os.Getenv("JAVA2GO_SOURCE_STORAGE_PREBUILT_CLI")
	if binary == "" {
		return nil
	}
	want := os.Getenv("JAVA2GO_SOURCE_STORAGE_PREBUILT_SHA256")
	if len(want) != 64 {
		t.Fatal("declared CLI build digest required")
	}
	owner := &campaignCompilerParentCLI{directory: t.TempDir()}
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	used, reused, err := owner.executable(repo, goTool, append(os.Environ(), "GOWORK=off"), func(output string) error {
		data, err := os.ReadFile(binary)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != want {
			return fmt.Errorf("declared CLI bytes changed")
		}
		return os.WriteFile(output, data, 0755)
	})
	if err != nil || reused {
		t.Fatalf("bind exact declared CLI: reused=%v error=%v", reused, err)
	}
	data, err := os.ReadFile(used)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != want {
		t.Fatal("owned CLI differs from declared prebuild")
	}
	t.Logf("declared CLI consumed unchanged: SHA256=%s source/tool/environment bound by parent owner", want)
	return []*campaignCompilerParentCLI{owner}
}

func TestGenericFamilySourceInterfaceStorageImplementationsJDK21(t *testing.T) {
	source, err := os.ReadFile("testdata/source_interface_storage_implementations/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>source-interface-implementations</artifactId><version>1</version></project>`,
		"src/main/java/sourceinterfaceimplementations/Main.java": string(source),
	}, "sourceinterfaceimplementations.Main", "ok|true|true|true|true|true|true|true\n", sourceInterfaceStorageParentCLI(t)...)
}
