package transpiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassIsInstanceArgumentBoxing63(t *testing.T) {
	const source = `public class InstanceBoxing63 { public static boolean run() { return java.lang.Integer.class.isInstance(3); } }`
	generated := renderGoFileFromJava(t, source)
	if !strings.Contains(generated, ".IsInstance(") || !strings.Contains(generated, "BoxInteger(") {
		t.Fatalf("Class.isInstance must invoke nominal runtime check after Object boxing:\n%s", generated)
	}
}

func TestClassIsInstanceJDKProject63(t *testing.T) {
	root := filepath.Join("testdata", "class_isinstance63")
	source, err := os.ReadFile(filepath.Join(root, "src/main/java/app/Main.java"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "oracle.stdout"))
	if err != nil {
		t.Fatalf("independent JDK capture required before generated comparison: %v", err)
	}
	stderr, err := os.ReadFile(filepath.Join(root, "oracle.stderr"))
	if err != nil || len(stderr) != 0 {
		t.Fatalf("independent JDK stderr must exist and be empty: %v", err)
	}
	files := map[string]string{"pom.xml": `<project><groupId>review</groupId><artifactId>class-isinstance63</artifactId><version>1</version></project>`, "src/main/java/app/Main.java": string(source)}
	runCampaignCompilerStrictProjectOracle47Args(t, files, "app.Main", string(want))
}
