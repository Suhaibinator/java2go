package transpiler

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestRAFStringContractsWholeProgramJVM(t *testing.T)  { rafStringFrozenFixture(t, "contracts") }
func TestRAFStringOpenErrorsWholeProgramJVM(t *testing.T) { rafStringFrozenFixture(t, "open-errors") }
func TestRAFStringOrderingWholeProgramJVM(t *testing.T)   { rafStringFrozenFixture(t, "ordering") }
func TestRAFStringOwnersWholeProgramJVM(t *testing.T)     { rafStringFrozenFixture(t, "owners") }
func rafStringFrozenFixture(t *testing.T, name string) {
	t.Helper()
	fixture := filepath.Join("..", "campaign", "reproducers", "jdk-random-access-file-string", name)
	root := t.TempDir()
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	var sources []string
	err = filepath.WalkDir(fixture, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(fixture, path)
		if err != nil {
			return err
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(target, contents, 0600); err != nil {
			return err
		}
		if filepath.Ext(path) == ".java" {
			sources = append(sources, target)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	setup, err := os.ReadFile(filepath.Join(root, "fixture.setup.json"))
	if err == nil {
		var config struct {
			FileModes map[string]uint32 `json:"file_modes"`
		}
		if err := json.Unmarshal(setup, &config); err != nil {
			t.Fatal(err)
		}
		for path, mode := range config.FileModes {
			if err := os.Chmod(filepath.Join(root, path), os.FileMode(mode)); err != nil {
				t.Fatal(err)
			}
		}
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(sources) == 0 {
		t.Fatal("zero-selected source fixture")
	}
	sort.Strings(sources)
	run := func(stage, dir string, limit time.Duration, command string, args ...string) ([]byte, []byte) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), limit)
		defer cancel()
		cmd := exec.CommandContext(ctx, command, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOMAXPROCS=2", "GOFLAGS=-p=1")
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		cmd.WaitDelay = 5 * time.Second
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s: %v stdout=%s stderr=%s", stage, err, stdout.Bytes(), stderr.Bytes())
		}
		return stdout.Bytes(), stderr.Bytes()
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
	run("javac", root, 55*time.Second, javac, append([]string{"--release", "21", "-encoding", "UTF-8", "-d", classes}, sources...)...)
	expected, err := os.ReadFile(filepath.Join(fixture, "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	expectedErr, err := os.ReadFile(filepath.Join(fixture, "expected.stderr"))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		out, errout := run("JDK oracle", root, 60*time.Second, java, "-cp", classes, "probe.Main")
		if !bytes.Equal(out, expected) || !bytes.Equal(errout, expectedErr) {
			t.Fatalf("frozen JDK oracle changed stdout=%q stderr=%q", out, errout)
		}
	}
	compiler := filepath.Join(root, "java2go")
	run("compiler build", repo, 300*time.Second, "go", "build", "-o", compiler, "./cmd/java2go")
	generated := filepath.Join(root, "generated")
	run("strict", repo, 60*time.Second, compiler, "-strict", "-maven", root, "-main-class", "probe.Main", "-runtime", repo, "-output", generated)
	run("all generated packages race", generated, 300*time.Second, "go", "build", "-race", "-mod=mod", "./...")
	binary := filepath.Join(root, "app")
	run("entry race", generated, 300*time.Second, "go", "build", "-race", "-mod=mod", "-o", binary, "./cmd/app")
	for i := 0; i < 3; i++ {
		out, errout := run("Go observation", root, 60*time.Second, binary)
		if !bytes.Equal(out, expected) || !bytes.Equal(errout, expectedErr) {
			t.Fatalf("Go/JDK difference stdout=%q stderr=%q; expected=%q", out, errout, expected)
		}
	}
}

func TestRAFStringIntrinsicStrictRefusal(t *testing.T) {
	for _, source := range []string{
		`import java.io.RandomAccessFile;class Probe{Object run()throws Exception{return new RandomAccessFile(null,"r");}}`,
		`import java.io.RandomAccessFile;class Probe{Object run()throws Exception{return new RandomAccessFile((java.io.File)null,"r");}}`,
		`import java.io.RandomAccessFile;class Probe{Object run()throws Exception{return new RandomAccessFile("path",7);}}`,
		`import java.io.RandomAccessFile;class Probe{Object run()throws Exception{return new RandomAccessFile("path");}}`,
		`import java.io.RandomAccessFile;class Probe<RandomAccessFile>{Object run()throws Exception{return new RandomAccessFile("path","r");}}`,
		`import foreign.RandomAccessFile;class Probe{Object run()throws Exception{return new RandomAccessFile("path","r");}}`,
		`import java.io.RandomAccessFile;class Probe{Object run()throws Exception{return new RandomAccessFile<Object>("path","r");}}`,
		`import java.io.RandomAccessFile;class String{}class Probe{Object run(String s)throws Exception{return new RandomAccessFile(s,"r");}}`,
	} {
		t.Run(source, func(t *testing.T) {
			strictRoutingState(t)
			defer func() {
				if recover() == nil || len(Diagnostics()) == 0 {
					t.Fatal("invalid constructor escaped strict refusal")
				}
			}()
			renderGoFileFromJava(t, source)
		})
	}
}
func TestRAFStringSourceDeclarationOwnership(t *testing.T) {
	source := `class RandomAccessFile{RandomAccessFile(String path,String mode){}}class Probe{static Object run(){return new RandomAccessFile("source","mode");}}`
	generated := renderGoFileFromJava(t, source)
	if strings.Contains(generated, "NewRandomAccessFileStringExecution") || strings.Contains(generated, "stdjava.RandomAccessFile") {
		t.Fatal("source constructor borrowed JDK service", generated)
	}
}

func TestRAFStringReadMessageIdentityWholeProgramJVM(t *testing.T) {
	readonlyFrozenFixture(t, "read-message-identity")
}
