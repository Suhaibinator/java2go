package transpiler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCampaignEmbeddedClassResourcesJVMParity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	root := t.TempDir()
	repo, _ := filepath.Abs("..")
	generated := filepath.Join(root, "generated")
	files := map[string]string{
		"pom.xml":                         `<project><groupId>example</groupId><artifactId>resources</artifactId><version>1</version></project>`,
		"src/main/java/example/Main.java": `package example;import java.io.*;import java.nio.charset.StandardCharsets;public class Main{public static void main(String[]args)throws Exception{try(InputStream a=Main.class.getResourceAsStream("/root.txt");InputStream b=Main.class.getResourceAsStream("nested.txt")){System.out.println(new String(a.readAllBytes(),StandardCharsets.UTF_8)+":"+new String(b.readAllBytes(),StandardCharsets.UTF_8)+":"+(Main.class.getResourceAsStream("/missing")==null)+":"+(Main.class.getResourceAsStream("/payload.go")!=null)+":"+(Main.class.getResourceAsStream("/go.mod")==null));}}}`,
		"src/main/resources/payload.go":   "not Go source",
		"src/main/resources/root.txt":     "embedded", "src/main/resources/example/nested.txt": "relative",
	}
	for path, data := range files {
		if err := writeProjectFile(filepath.Join(root, path), []byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	java, err := campaignCompilerJavaTool("java")
	if err != nil {
		t.Fatal(err)
	}
	classes := filepath.Join(root, "classes")
	if out, err := exec.CommandContext(ctx, javac, "--release", "21", "-d", classes, filepath.Join(root, "src/main/java/example/Main.java")).CombinedOutput(); err != nil {
		t.Fatalf("javac: %v\n%s", err, out)
	}
	cwd := filepath.Join(root, "unrelated")
	if err := writeProjectFile(filepath.Join(cwd, "resources/root.txt"), []byte("wrong-cwd")); err != nil {
		t.Fatal(err)
	}
	oracle := exec.CommandContext(ctx, java, "-cp", classes+string(os.PathListSeparator)+filepath.Join(root, "src/main/resources"), "example.Main")
	oracle.Dir = cwd
	want, err := oracle.CombinedOutput()
	if err != nil {
		t.Fatalf("java: %v\n%s", err, want)
	}
	compiler := exec.CommandContext(ctx, "go", "run", "./cmd/java2go", "-maven", root, "-main-class", "example.Main", "-runtime", repo, "-output", generated)
	compiler.Dir = repo
	if out, err := compiler.CombinedOutput(); err != nil {
		t.Fatalf("compiler: %v\n%s", err, out)
	}
	all := exec.CommandContext(ctx, "go", "build", "-mod=mod", "./...")
	all.Dir = generated
	all.Env = append(os.Environ(), "GOWORK=off")
	if out, err := all.CombinedOutput(); err != nil {
		t.Fatalf("build all: %v\n%s", err, out)
	}
	binary := filepath.Join(root, "app")
	build := exec.CommandContext(ctx, "go", "build", "-mod=mod", "-o", binary, "./cmd/app")
	build.Dir = generated
	build.Env = append(os.Environ(), "GOWORK=off")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	// Remove both copied and original resource trees before launching elsewhere.
	for _, path := range []string{filepath.Join(root, "src/main/resources"), filepath.Join(generated, "resources")} {
		if err := os.RemoveAll(path); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.CommandContext(ctx, binary)
	command.Dir = cwd
	got, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, got)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("JVM %q != Go %q", want, got)
	}
}
