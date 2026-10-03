package transpiler

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMavenFailedConversionRetainsPrivateArtifacts(t *testing.T) {
	root := t.TempDir()
	runtimeRoot, _ := filepath.Abs("..")
	files := map[string]string{"pom.xml": `<project><groupId>example</groupId><artifactId>retention</artifactId><version>1</version></project>`, "src/main/java/example/Main.java": `package example;public class Main{public static int value(){return 42;}}`}
	for path, data := range files {
		if err := writeProjectFile(filepath.Join(root, path), []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	output := filepath.Join(root, "generated")
	err := run([]string{"-maven", root, "-main-class", "example.Main", "-runtime", runtimeRoot, "-output", output}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "must declare public static void main") {
		t.Fatalf("expected entrypoint failure, got %v", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("failed output published: %v", statErr)
	}
	stages, globErr := filepath.Glob(filepath.Join(root, ".java2go-project-*"))
	if globErr != nil || len(stages) != 1 {
		t.Fatalf("failed staging not retained: %v %v", stages, globErr)
	}
	if !strings.Contains(err.Error(), stages[0]) {
		t.Fatalf("error omits retained path: %v", err)
	}
	info, statErr := os.Stat(stages[0])
	if statErr != nil {
		t.Fatal(statErr)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatalf("staging permissions %v", info.Mode())
	}
	for _, path := range []string{"sources/j_example/Main.java", "output/j_example/Main.go"} {
		data, readErr := os.ReadFile(filepath.Join(stages[0], path))
		if readErr != nil || len(data) == 0 {
			t.Fatalf("lost diagnostic artifact %s: %v", path, readErr)
		}
	}
}
func TestMavenSuccessfulConversionCleansStaging(t *testing.T) {
	root := t.TempDir()
	runtimeRoot, _ := filepath.Abs("..")
	for path, data := range map[string]string{"pom.xml": `<project><groupId>example</groupId><artifactId>retention</artifactId><version>1</version></project>`, "src/main/java/example/Main.java": `package example;public class Main{public static void main(String[]args){System.out.println(42);}}`} {
		if err := writeProjectFile(filepath.Join(root, path), []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	output := filepath.Join(root, "generated")
	if err := run([]string{"-maven", root, "-main-class", "example.Main", "-runtime", runtimeRoot, "-output", output}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(output, "cmd/app/main.go")); err != nil {
		t.Fatal(err)
	}
	stages, err := filepath.Glob(filepath.Join(root, ".java2go-project-*"))
	if err != nil || len(stages) != 0 {
		t.Fatalf("successful staging leaked: %v %v", stages, err)
	}
}

func TestMavenPanicUnwindingRetainsArtifacts(t *testing.T) {
	staging := t.TempDir()
	artifact := filepath.Join(staging, "diagnostic.go")
	if err := os.WriteFile(artifact, []byte("package diagnostic\n"), 0600); err != nil {
		t.Fatal(err)
	}
	marker := &struct{ label string }{"original panic"}
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		var result error
		defer finishMavenProjectStaging(staging, false, &result)
		panic(marker)
	}()
	if recovered != marker {
		t.Fatalf("panic changed: %v", recovered)
	}
	if _, err := os.Stat(artifact); err != nil {
		t.Fatalf("panic removed staged artifact: %v", err)
	}
}
