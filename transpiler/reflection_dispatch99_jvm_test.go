package transpiler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestReflectionDispatch99JDK21(t *testing.T) {
	reflectionDispatch99ProjectJDK21Main(t, "reflection_dispatch99", "dispatch.app.Main")
}
func reflectionDispatch99ProjectJDK21Main(t *testing.T, fixture, mainClass string) {
	run := func(label, dir string, limit time.Duration, name string, args ...string) ([]byte, []byte) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), limit)
		defer cancel()
		switch filepath.Base(name) {
		case "java":
			args = append([]string{"-XX:ActiveProcessorCount=2"}, args...)
		case "javac":
			args = append([]string{"-J-XX:ActiveProcessorCount=2"}, args...)
		}
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOMAXPROCS=2", "GOFLAGS=-p=1")
		cmd.WaitDelay = 5 * time.Second
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		t.Logf("%s: %v", label, cmd.Args)
		if ctx.Err() != nil || err != nil {
			t.Fatalf("%s: %v, deadline=%v\nstdout:%s\nstderr:%s", label, err, ctx.Err(), stdout.Bytes(), stderr.Bytes())
		}
		return stdout.Bytes(), stderr.Bytes()
	}
	javac, java := methodReflectionJavaTool(t, "javac"), methodReflectionJavaTool(t, "java")
	version, _ := run("JDK version", ".", time.Minute, javac, "-version")
	if !strings.HasPrefix(string(version), "javac 21") {
		t.Fatalf("JDK21 required, got %q", version)
	}
	root := t.TempDir()
	var sources []string
	err := filepath.WalkDir(filepath.Join("testdata", fixture, "project"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(filepath.Join("testdata", fixture, "project"), path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		destination := filepath.Join(root, rel)
		if err := writeProjectFile(destination, content); err != nil {
			return err
		}
		if filepath.Ext(path) == ".java" {
			sources = append(sources, destination)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(sources)
	classes := filepath.Join(root, "classes")
	run("javac", root, 5*time.Minute, javac, append([]string{"--release", "21", "-encoding", "UTF-8", "-d", classes}, sources...)...)
	want, wantErr := run("live JVM", root, time.Minute, java, "-cp", classes, mainClass)
	t.Logf("live JVM stdout=%q stderr=%q", want, wantErr)
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	compiler := filepath.Join(root, "java2go")
	run("compiler build", repo, 5*time.Minute, "go", "build", "-o", compiler, "./cmd/java2go")
	generated := filepath.Join(root, "generated")
	run("strict", repo, 5*time.Minute, compiler, "-strict", "-maven", root, "-main-class", mainClass, "-runtime", repo, "-output", generated)
	run("all generated packages", generated, 5*time.Minute, "go", "build", "-mod=mod", "./...")
	app := filepath.Join(root, "app")
	run("entry build", generated, 5*time.Minute, "go", "build", "-mod=readonly", "-o", app, "./cmd/app")
	got, gotErr := run("generated runtime", root, time.Minute, app)
	if !bytes.Equal(got, want) || !bytes.Equal(gotErr, wantErr) {
		t.Fatalf("generated stdout=%q stderr=%q != actual JVM stdout=%q stderr=%q", got, gotErr, want, wantErr)
	}
}
