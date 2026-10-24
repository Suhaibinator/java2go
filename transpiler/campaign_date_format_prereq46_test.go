package transpiler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Preserve the unchanged default-package JDK probe through the strict CLI file
// path. Maven projects require named packages and cannot host this exact input.
func TestCampaignDateFormatPrereq46(t *testing.T) {
	fixture := filepath.Join("testdata", "date_format_prereq46")
	source, err := os.ReadFile(filepath.Join(fixture, "src/DateFormatSequence.java"))
	if err != nil {
		t.Fatal(err)
	}
	observed, err := os.ReadFile(filepath.Join(fixture, "oracle.stdout"))
	if err != nil {
		t.Fatalf("independent JVM oracle is required before runtime implementation: %v", err)
	}
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	java, err := campaignCompilerJavaTool("java")
	if err != nil {
		t.Fatal(err)
	}
	goTool := os.Getenv("JAVA2GO_TIME_GATE_GO")
	if goTool == "" {
		goTool = "go"
	}
	run := func(stage, directory string, bound time.Duration, program string, args ...string) ([]byte, []byte) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), bound)
		defer cancel()
		cmd := exec.CommandContext(ctx, program, args...)
		cmd.Dir = directory
		cmd.Env = append(os.Environ(), "GOWORK=off")
		cmd.WaitDelay = 5 * time.Second
		var out, errOut bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &errOut
		started := time.Now()
		runErr := cmd.Run()
		exitCode := 0
		if runErr != nil {
			exitCode = -1
			if e, ok := runErr.(*exec.ExitError); ok {
				exitCode = e.ExitCode()
			}
		}
		if artifacts := os.Getenv("JAVA2GO_TIME_GATE_ARTIFACTS"); artifacts != "" {
			if err := os.MkdirAll(artifacts, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(artifacts, stage+".stdout"), out.Bytes(), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(artifacts, stage+".stderr"), errOut.Bytes(), 0644); err != nil {
				t.Fatal(err)
			}
			data, err := json.MarshalIndent(map[string]any{"stage": stage, "command": cmd.Args, "cwd": directory, "exit_code": exitCode, "timed_out": ctx.Err() != nil, "elapsed_seconds": time.Since(started).Seconds()}, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(artifacts, stage+".json"), append(data, '\n'), 0644); err != nil {
				t.Fatal(err)
			}
		}
		t.Logf("%s: %v (%s)", stage, cmd.Args, time.Since(started))
		if ctx.Err() != nil {
			t.Fatalf("%s timed out: %v\nstdout: %s\nstderr: %s", stage, ctx.Err(), out.Bytes(), errOut.Bytes())
		}
		if runErr != nil {
			t.Fatalf("%s: %v\nstdout: %s\nstderr: %s", stage, runErr, out.Bytes(), errOut.Bytes())
		}
		return out.Bytes(), errOut.Bytes()
	}
	root := t.TempDir()
	input := filepath.Join(root, "DateFormatSequence.java")
	if err := writeProjectFile(input, source); err != nil {
		t.Fatal(err)
	}
	classes := filepath.Join(root, "classes")
	emptyInput := filepath.Join(root, "empty-input-path")
	if err := os.Mkdir(emptyInput, 0755); err != nil {
		t.Fatal(err)
	}
	run("javac", root, 5*time.Minute, javac, "--release", "21", "-encoding", "UTF-8", "-proc:none", "-classpath", emptyInput, "-sourcepath", emptyInput, "-d", classes, input)
	compiledHashes := map[string]string{}
	if err := filepath.WalkDir(classes, func(path string, item os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if item.IsDir() || filepath.Ext(path) != ".class" {
			return nil
		}
		relative, err := filepath.Rel(classes, path)
		if err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		compiledHashes[relative] = fmt.Sprintf("%x", sha256.Sum256(raw))
		if artifacts := os.Getenv("JAVA2GO_TIME_GATE_ARTIFACTS"); artifacts != "" {
			return writeProjectFile(filepath.Join(artifacts, "classes", relative), raw)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if artifacts := os.Getenv("JAVA2GO_TIME_GATE_ARTIFACTS"); artifacts != "" {
		record, err := json.MarshalIndent(compiledHashes, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := writeProjectFile(filepath.Join(artifacts, "classes-hashes.json"), append(record, '\n')); err != nil {
			t.Fatal(err)
		}
	}
	want, wantErr := run("JVM", root, time.Minute, java, "-Dfile.encoding=UTF-8", "-Dstdout.encoding=UTF-8", "-Dstderr.encoding=UTF-8", "-Duser.language=en", "-Duser.country=US", "-Duser.timezone=UTC", "-cp", classes, "DateFormatSequence")
	if !bytes.Equal(want, observed) || len(wantErr) != 0 {
		t.Fatalf("invalid JVM oracle: stdout=%q stderr=%q; expected stdout=%q and empty stderr", want, wantErr, observed)
	}
	t.Logf("JVM oracle: %q", want)
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	compiler := filepath.Join(root, "java2go")
	run("compiler-build", repo, 5*time.Minute, goTool, "build", "-o", compiler, "./cmd/java2go")
	generated := filepath.Join(root, "generated")
	run("strict-transpilation", repo, 5*time.Minute, compiler, "-strict", "-sync", "-w", "-output", generated, input)
	// The single-file CLI exposes Java's public main as Main(), a fresh
	// execution wrapper (buildExecutionAwareFuncDecls), as used by
	// TestCampaignLocalTypeShadowOriginalJVMParity. Add only the Go entrypoint;
	// the frozen Java and every generated Go byte remain unchanged.
	generatedFile := filepath.Join(generated, "DateFormatSequence.go")
	generatedBytes, err := os.ReadFile(generatedFile)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), generatedFile, generatedBytes, 0)
	if err != nil {
		t.Fatal(err)
	}
	entryCount := 0
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != "Main" {
			continue
		}
		entryCount++
		if function.Recv != nil || (function.Type.Params != nil && function.Type.Params.NumFields() != 0) || (function.Type.Results != nil && function.Type.Results.NumFields() != 0) {
			t.Fatal("generated public Main wrapper has unexpected signature")
		}
	}
	if entryCount != 1 {
		t.Fatalf("want one generated public Main wrapper, got %d", entryCount)
	}
	launcher := []byte("package main\n\nfunc main() { Main() }\n")
	if err := writeProjectFile(filepath.Join(generated, "entrypoint.go"), launcher); err != nil {
		t.Fatal(err)
	}
	if artifacts := os.Getenv("JAVA2GO_TIME_GATE_ARTIFACTS"); artifacts != "" {
		if err := writeProjectFile(filepath.Join(artifacts, "generated", "DateFormatSequence.go"), generatedBytes); err != nil {
			t.Fatal(err)
		}
		if err := writeProjectFile(filepath.Join(artifacts, "entrypoint.go"), launcher); err != nil {
			t.Fatal(err)
		}
		record, err := json.MarshalIndent(map[string]any{"generated_source_sha256": fmt.Sprintf("%x", sha256.Sum256(generatedBytes)), "launcher_sha256": fmt.Sprintf("%x", sha256.Sum256(launcher)), "generated_source_unchanged": true, "launcher_only_invokes_generated_Main": true, "Main_is_fresh_execution_wrapper": true}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := writeProjectFile(filepath.Join(artifacts, "launcher.json"), append(record, '\n')); err != nil {
			t.Fatal(err)
		}
	}

	module := "module timeprobe\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => " + repo + "\n"
	if err := writeProjectFile(filepath.Join(generated, "go.mod"), []byte(module)); err != nil {
		t.Fatal(err)
	}
	run("all-generated-race-build", generated, 5*time.Minute, goTool, "build", "-race", "-mod=mod", "./...")
	binary := filepath.Join(root, "generated-app")
	run("generated-entry-race-build", generated, 5*time.Minute, goTool, "build", "-race", "-mod=mod", "-o", binary, ".")
	got, gotErr := run("generated-Go", root, time.Minute, binary)
	if !bytes.Equal(got, want) || !bytes.Equal(gotErr, wantErr) {
		t.Fatalf("Go stdout=%q stderr=%q differs from JVM stdout=%q stderr=%q", got, gotErr, want, wantErr)
	}
}
