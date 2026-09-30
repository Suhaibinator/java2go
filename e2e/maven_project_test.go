package e2e

import (
	"bytes"
	"context"
	java2go "github.com/NickyBoy89/java2go"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// This application crosses reactor and external-source dependency boundaries,
// consumes a production resource, and observes Java's command-line arguments.
func TestMavenApplicationParity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	javac, err := exec.LookPath("javac")
	if err != nil {
		t.Skip("javac is required for Maven application parity")
	}
	java, err := exec.LookPath("java")
	if err != nil {
		t.Skip("java is required for Maven application parity")
	}
	fixture, _ := filepath.Abs("../testfiles/maven_application")
	repo, _ := filepath.Abs("..")
	classes := t.TempDir()
	sources := []string{}
	for _, root := range []string{"reactor/app/src/main/java", "reactor/domain/src/main/java", "vendor/src/main/java"} {
		err := filepath.WalkDir(filepath.Join(fixture, root), func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && filepath.Ext(path) == ".java" {
				sources = append(sources, path)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	command := exec.CommandContext(ctx, javac, append([]string{"-d", classes}, sources...)...)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("javac: %v\n%s", err, out)
	}
	if err := os.Mkdir(filepath.Join(classes, "resources"), 0755); err != nil {
		t.Fatal(err)
	}
	resource, err := os.ReadFile(filepath.Join(fixture, "reactor/app/src/main/resources/banner.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(classes, "resources/banner.txt"), resource, 0644); err != nil {
		t.Fatal(err)
	}
	command = exec.CommandContext(ctx, java, "-cp", classes, "example.app.InvoiceApp", "customer")
	command.Dir = classes
	var javaStderr bytes.Buffer
	command.Stderr = &javaStderr
	want, err := command.Output()
	if err != nil {
		t.Fatalf("java: %v\n%s", err, want)
	}
	if string(want) != "customer:42\ninvoice-ready\n" {
		t.Fatalf("unexpected Java oracle: %q", want)
	}
	t.Logf("Java oracle: %q", want)
	output := filepath.Join(t.TempDir(), "generated")
	if err := java2go.Run([]string{"-maven", filepath.Join(fixture, "reactor"), "-dependency-source", "vendor:formatter=" + filepath.Join(fixture, "vendor"), "-main-class", "example.app.InvoiceApp", "-runtime", repo, "-module", "example.test/invoice", "-output", output}); err != nil {
		t.Fatal(err)
	}
	command = exec.CommandContext(ctx, "go", "run", "-mod=mod", "./cmd/app", "customer")
	command.Dir = output
	var goStderr bytes.Buffer
	command.Stderr = &goStderr
	got, err := command.Output()
	if err != nil {
		t.Fatalf("generated Go: %v\n%s", err, goStderr.String())
	}
	if javaStderr.Len() != 0 || goStderr.Len() != 0 {
		t.Fatalf("unexpected stderr: Java %q; Go %q", javaStderr.String(), goStderr.String())
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("parity mismatch: Go %q, Java %q", got, want)
	}
	t.Logf("Generated Go matches Java: %q", got)
	if _, err := os.Stat(filepath.Join(output, "j_example/j_app/TestOnly.go")); !os.IsNotExist(err) {
		t.Fatal("test-only source was included")
	}
}
