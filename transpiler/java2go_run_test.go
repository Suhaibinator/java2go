package transpiler

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeJavaSource(t *testing.T, root string, relativePath string, source string) string {
	t.Helper()

	fullPath := filepath.Join(root, relativePath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte(source), 0644); err != nil {
		t.Fatalf("failed to write java source: %v", err)
	}
	return fullPath
}

func TestRun_InitGoMod_WritesModuleFile(t *testing.T) {
	inputDir := filepath.Join(t.TempDir(), "input")
	writeJavaSource(t, inputDir, "com/example/App.java", `
package com.example;
public class App {
    public static void main(String[] args) {
        System.out.println("hello");
    }
}
`)

	outputDir := filepath.Join(t.TempDir(), "output")
	err := run([]string{
		"-w",
		"-output", outputDir,
		"-init-go-mod",
		"-module", "example.com/generated/app",
		inputDir,
	}, io.Discard)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	goModPath := filepath.Join(outputDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("expected go.mod at %s: %v", goModPath, err)
	}

	flat := string(content)
	if !strings.Contains(flat, "module example.com/generated/app") {
		t.Fatalf("expected module path in go.mod, got:\n%s", flat)
	}
	if !strings.Contains(flat, "go 1.27.0") {
		t.Fatalf("expected go version in go.mod, got:\n%s", flat)
	}
}

func TestRun_InitGoMod_DoesNotOverwriteExistingFile(t *testing.T) {
	inputDir := filepath.Join(t.TempDir(), "input")
	writeJavaSource(t, inputDir, "Main.java", `
public class Main {}
`)

	outputDir := filepath.Join(t.TempDir(), "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("failed creating output dir: %v", err)
	}

	existing := "module keep/me\n\ngo 1.27.0\n"
	goModPath := filepath.Join(outputDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(existing), 0644); err != nil {
		t.Fatalf("failed writing existing go.mod: %v", err)
	}

	err := run([]string{
		"-w",
		"-output", outputDir,
		"-init-go-mod",
		"-module", "example.com/other",
		inputDir,
	}, io.Discard)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	got, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("failed reading go.mod: %v", err)
	}
	if string(got) != existing {
		t.Fatalf("expected existing go.mod content to remain unchanged, got:\n%s", string(got))
	}
}

func TestRun_ModuleRelativeLayoutForMatchingJavaPackage(t *testing.T) {
	inputDir := filepath.Join(t.TempDir(), "input")
	writeJavaSource(t, inputDir, "com/acme/app/MainApp.java", `
package com.acme.app;
public class MainApp {
    public static void main(String[] args) {}
}
`)

	outputDir := filepath.Join(t.TempDir(), "output")
	err := run([]string{
		"-w",
		"-output", outputDir,
		"-init-go-mod",
		"-module", "com/acme",
		inputDir,
	}, io.Discard)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	expectedPath := filepath.Join(outputDir, "app", "MainApp.go")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected module-relative output file at %s: %v", expectedPath, err)
	}
}
