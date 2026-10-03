package transpiler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// Resolve an installed JDK21 even when JAVA_HOME is absent. Expectations come
// from this live JDK process, never a generated-Go observation or raw gate.
func methodReflectionJavaTool(t *testing.T, name string) string {
	t.Helper()
	home := os.Getenv("JAVA_HOME")
	if home == "" && runtime.GOOS == "darwin" {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		out, err := exec.CommandContext(ctx, "/usr/libexec/java_home", "-v", "21").Output()
		if err != nil {
			t.Fatalf("installed JDK21 discovery: %v", err)
		}
		home = strings.TrimSpace(string(out))
	}
	tool := name
	if home != "" {
		tool = filepath.Join(home, "bin", name)
	}
	resolved, err := exec.LookPath(tool)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestMethodReflectionSourceBridgesJDK21(t *testing.T) {
	for _, fixture := range []string{"method_reflection92", "method_reflection_edges92", "method_reflection_class_flags92", "method_reflection_modifiers92", "method_reflection_inherited_bridge94", "method_reflection_inherited_stateful94"} {
		t.Run(fixture, func(t *testing.T) { methodReflectionProjectJDK21(t, fixture) })
	}
}

func methodReflectionProjectJDK21(t *testing.T, fixture string) {
	methodReflectionProjectJDK21Main(t, fixture, "reflection.app.Main")
}

func methodReflectionProjectJDK21Main(t *testing.T, fixture, mainClass string) {
	run := func(label, dir string, limit time.Duration, name string, args ...string) ([]byte, []byte) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), limit)
		defer cancel()
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
