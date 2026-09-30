package transpiler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// Seeded additive regression transport preserves independent Java inputs and
// passes exactly the same application arguments to JVM and generated Go.
// Original strict/race/build/parity requirements remain unchanged.
func runCampaignCompilerStrictProjectOracle47Args(t *testing.T, files map[string]string, mainClass, expected string, applicationArgs ...string) {
	t.Helper()
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	java, err := campaignCompilerJavaTool("java")
	if err != nil {
		t.Fatal(err)
	}
	run := func(stage, dir string, limit time.Duration, name string, args ...string) ([]byte, []byte) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), limit)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off")
		cmd.WaitDelay = 5 * time.Second
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		started := time.Now()
		err := cmd.Run()
		t.Logf("%s: %v (%s)", stage, cmd.Args, time.Since(started))
		if ctx.Err() != nil {
			t.Fatalf("%s timed out: %v\nstdout: %s\nstderr: %s", stage, ctx.Err(), stdout.Bytes(), stderr.Bytes())
		}
		if err != nil {
			t.Fatalf("%s: %v\nstdout: %s\nstderr: %s", stage, err, stdout.Bytes(), stderr.Bytes())
		}
		return stdout.Bytes(), stderr.Bytes()
	}
	root := t.TempDir()
	var sources []string
	for name, source := range files {
		path := filepath.Join(root, name)
		if err := writeProjectFile(path, []byte(source)); err != nil {
			t.Fatal(err)
		}
		if filepath.Ext(name) == ".java" {
			sources = append(sources, path)
		}
	}
	sort.Strings(sources)
	classes := filepath.Join(root, "classes")
	run("javac", root, 5*time.Minute, javac, append([]string{"--release", "21", "-encoding", "UTF-8", "-d", classes}, sources...)...)
	want, wantErr := run("JVM", root, time.Minute, java, append([]string{"-cp", classes, mainClass}, applicationArgs...)...)
	if string(want) != expected || len(wantErr) != 0 {
		t.Fatalf("invalid JVM oracle: stdout=%q stderr=%q; expected stdout=%q and empty stderr", want, wantErr, expected)
	}
	t.Logf("JVM oracle: %q", want)
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	compiler := filepath.Join(root, "java2go")
	run("compiler build", repo, 5*time.Minute, "go", "build", "-o", compiler, "./cmd/java2go")
	generated := filepath.Join(root, "generated")
	run("strict transpilation", repo, 5*time.Minute, compiler, "-strict", "-maven", root, "-main-class", mainClass, "-runtime", repo, "-output", generated)
	run("all generated packages race build", generated, 5*time.Minute, "go", "build", "-race", "-mod=mod", "./...")
	binary := filepath.Join(root, "generated-app")
	run("generated entry point race build", generated, 5*time.Minute, "go", "build", "-race", "-mod=mod", "-o", binary, "./cmd/app")
	got, gotErr := run("generated Go", root, time.Minute, binary, applicationArgs...)
	t.Logf("generated Go streams: stdout=%q stderr=%q", got, gotErr)
	if !bytes.Equal(got, want) || !bytes.Equal(gotErr, wantErr) {
		t.Fatalf("Go stdout=%q stderr=%q differs from JVM stdout=%q stderr=%q", got, gotErr, want, wantErr)
	}
}
