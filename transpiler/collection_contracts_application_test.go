package transpiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The checked-in oracle keeps generated-code parity mandatory without a JDK;
// when Java is installed this test also verifies the oracle against javac/java.
func TestCollectionContractsApplication(t *testing.T) {
	for _, name := range []string{"CollectionContracts", "CollectionValues"} {
		t.Run(name, func(t *testing.T) {
			runCollectionApplication(t, name)
		})
	}
}

func runCollectionApplication(t *testing.T, name string) {
	directory := "java_collection_contracts"
	if name == "CollectionValues" {
		directory = "java_collection_values"
	}
	root := filepath.Join("..", "testfiles", "applications", directory)
	source, err := os.ReadFile(filepath.Join(root, "src", name+".java"))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(root, "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(expected))
	if javac, err := exec.LookPath("javac"); err == nil {
		dir := t.TempDir()
		path := filepath.Join(dir, name+".java")
		if err := os.WriteFile(path, source, 0600); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command(javac, "-d", dir, path).CombinedOutput(); err != nil {
			t.Fatalf("javac: %v\n%s", err, out)
		}
		out, err := exec.Command("java", "-cp", dir, "parity.collections."+name).CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != want {
			t.Fatalf("Java oracle: %v, got %q, want %q", err, out, want)
		}
	}
	generated := renderGoFileFromJava(t, strings.Replace(string(source), "package parity.collections;", "", 1))
	runGeneratedWithStdjava(t, generated, `package main
import "testing"
func TestApplication(t *testing.T) { if got := Run(); got != `+strconv.Quote(want)+` { t.Fatalf("got %q", got) } }
`)
}
