package transpiler

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMavenProjectRejectsCyclesWithoutPublishing(t *testing.T) {
	root := t.TempDir()
	runtimeRoot, _ := filepath.Abs("..")
	files := map[string]string{
		"pom.xml":                             `<project><groupId>example</groupId><artifactId>cycle</artifactId><version>1</version></project>`,
		"src/main/java/example/a/A.java":      `package example.a; import example.b.B; public class A { public static int value() { return B.value(); } }`,
		"src/main/java/example/b/B.java":      `package example.b; import example.a.A; public class B { public static int value() { return 42; } public static A create() { return new A(); } }`,
		"src/main/java/example/app/Main.java": `package example.app; import example.a.A; public class Main { public static void main(String[] args) { System.out.println(A.value()); } }`,
	}
	for path, data := range files {
		if err := writeProjectFile(filepath.Join(root, path), []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	output := filepath.Join(root, "out")
	err := run([]string{"-maven", root, "-main-class", "example.app.Main", "-runtime", runtimeRoot, "-output", output}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "example/a -> example/b -> example/a") {
		t.Fatalf("expected exact cycle diagnostic, got %v", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("failed conversion published output")
	}
}
func TestMavenProjectFailuresPreserveExistingOutput(t *testing.T) {
	root := t.TempDir()
	runtimeRoot, _ := filepath.Abs("..")
	fixture, _ := filepath.Abs("../testfiles/maven_application/reactor")
	output := filepath.Join(root, "out")
	if err := writeProjectFile(filepath.Join(output, "keep.txt"), []byte("untouched")); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"-maven", fixture, "-main-class", "example.app.InvoiceApp", "-runtime", runtimeRoot, "-output", output}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "absent or empty") {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(output, "keep.txt"))
	if err != nil || string(data) != "untouched" {
		t.Fatalf("existing output changed: %s %v", data, err)
	}
}
func TestMavenProjectRejectsResourceCollision(t *testing.T) {
	root := t.TempDir()
	runtimeRoot, _ := filepath.Abs("..")
	files := map[string]string{
		"pom.xml":                             `<project><groupId>example</groupId><artifactId>app</artifactId><version>1</version><build><resources><resource><directory>one</directory></resource><resource><directory>two</directory></resource></resources></build></project>`,
		"src/main/java/example/app/Main.java": `package example.app; public class Main { public static void main(String[] args) {} }`, "one/name.txt": "one", "two/name.txt": "two"}
	for path, data := range files {
		if err := writeProjectFile(filepath.Join(root, path), []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	err := run([]string{"-maven", root, "-main-class", "example.app.Main", "-runtime", runtimeRoot, "-output", filepath.Join(root, "out")}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "duplicate production resource") {
		t.Fatal(err)
	}
}

func TestMavenProjectDeterministicModule(t *testing.T) {
	fixture, _ := filepath.Abs("../testfiles/maven_application")
	runtimeRoot, _ := filepath.Abs("..")
	root := t.TempDir()
	snapshots := []map[string]string{}
	for _, name := range []string{"first", "second"} {
		output := filepath.Join(root, name)
		err := run([]string{"-maven", filepath.Join(fixture, "reactor"), "-dependency-source", "vendor:formatter=" + filepath.Join(fixture, "vendor"), "-main-class", "example.app.InvoiceApp", "-runtime", runtimeRoot, "-module", "example.test/invoice", "-output", output}, &bytes.Buffer{})
		if err != nil {
			t.Fatal(err)
		}
		snapshot := map[string]string{}
		err = filepath.WalkDir(output, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(output, path)
			if err != nil {
				return err
			}
			snapshot[relative] = string(data)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		snapshots = append(snapshots, snapshot)
	}
	if len(snapshots[0]) != len(snapshots[1]) {
		t.Fatal("different output inventory")
	}
	for path, data := range snapshots[0] {
		if snapshots[1][path] != data {
			t.Fatalf("nondeterministic generated file %s", path)
		}
	}
}

func TestMavenProjectRejectsAmbiguousStandardLibraryPackage(t *testing.T) {
	root := t.TempDir()
	runtimeRoot, _ := filepath.Abs("..")
	if err := writeProjectFile(filepath.Join(root, "pom.xml"), []byte(`<project><groupId>example</groupId><artifactId>app</artifactId><version>1</version></project>`)); err != nil {
		t.Fatal(err)
	}
	if err := writeProjectFile(filepath.Join(root, "src/main/java/fmt/Main.java"), []byte(`package fmt; public class Main { public static void main(String[] args) { System.out.println("hello"); } }`)); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"-maven", root, "-main-class", "fmt.Main", "-runtime", runtimeRoot, "-output", filepath.Join(root, "out")}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "conflicts with a Go standard-library import") {
		t.Fatal(err)
	}
}
