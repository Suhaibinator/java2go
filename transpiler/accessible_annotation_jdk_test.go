package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAccessibleAnnotationActualGsonMinimalJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "accessible_annotation_gson_minimal", "MinimalWitness")
}

func runAccessibleAnnotationFixture(t *testing.T, fixture, main string) {
	t.Helper()
	root := filepath.Join("testdata", fixture)
	files := map[string]string{}
	err := filepath.WalkDir(filepath.Join(root, "project"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(filepath.Join(root, "project"), path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(root, "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, files, main, string(expected))
}

func TestAccessibleAnonymousInheritedInitializationJDK21(t *testing.T) {
	pom, err := os.ReadFile(filepath.Join("testdata", "accessible_annotation_gson_minimal", "project", "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"pom.xml": string(pom), "src/main/java/AnonymousInitializationWitness.java": `
import java.lang.reflect.Field;
public class AnonymousInitializationWitness {
 static abstract class Base {
  static int initialized = initialize();
  static int initialize() { return 7; }
  abstract int value();
 }
 public static void main(String[] args) throws Exception {
  Base instance = new Base() { int value() { return Base.initialized; } };
  Field declaration = Base.class.getDeclaredField("initialized");
  System.out.println(instance.value() + ":" + declaration.getName());
 }
}`}
	runCampaignCompilerStrictProjectOracle(t, files, "AnonymousInitializationWitness", "7:initialized\n")
}

func TestAccessibleAnnotationPrimitiveContractsJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "accessible_annotation_primitives", "AnnotationPrimitiveWitness")
}
