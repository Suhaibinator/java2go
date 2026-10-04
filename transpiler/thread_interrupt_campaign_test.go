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

func TestThreadInterruptNativeWholeProgramJVM(t *testing.T) {
	threadInterruptFrozenFixture(t, "native")
}

// Genuine JDK state observer source remains a held admission frontier.
func TestThreadInterruptNativeStateWholeProgramJVM(t *testing.T) {
	threadInterruptFrozenFixture(t, "native-state-held")
}
func threadInterruptFrozenFixture(t *testing.T, name string) {
	t.Helper()
	fixture := filepath.Join("..", "campaign", "reproducers", "jdk-thread-interrupt", name)
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

func TestThreadInterruptEmitterResultApplicability(t *testing.T) {
	for _, c := range []struct {
		source string
		want   bool
	}{
		{`class Probe{java.lang.Thread t;void run(){t.interrupt();}}`, true},
		{`class Probe{Thread Thread;void run(){Thread.interrupt();}}`, true},
		{`class Probe{void run(){Thread.currentThread().interrupt();}}`, true},
		{`class Probe{Thread t;void run(){t.interrupt(7);}}`, false},
		{`class Probe{Thread[] t;void run(){t.interrupt();}}`, false},
		{`class Thread{void interrupt(){}}class Probe{Thread t;void run(){t.interrupt();}}`, false},
		{`import foreign.Thread;class Probe{Thread t;void run(){t.interrupt();}}`, false},
		{`class Probe<Thread extends java.lang.Thread>{Thread t;void run(){t.interrupt();}}`, false},
		{`class Child extends java.lang.Thread{public void interrupt(){}}class Probe{Child t;void run(){t.interrupt();}}`, false},
	} {
		t.Run(c.source, func(t *testing.T) {
			helper := setupParseHelper(t, c.source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			result, ok := inferIntrinsicMethodResultType(invocation, ctx, helper.File.Source)
			if ok != c.want || ok && result != "void" {
				t.Fatalf("result applicability %q/%t want void/%t", result, ok, c.want)
			}
			generated := renderGoFileFromJava(t, c.source)
			if c.want != strings.Contains(generated, "stdjava.ThreadInterruptExecution(") {
				t.Fatalf("emitter disagrees with applicability: %s", generated)
			}
		})
	}
}
func TestThreadInterruptInvalidSignatureStrictRefusal(t *testing.T) {
	for _, source := range []string{`class Probe{Thread t;void run(){t.interrupt(7);}}`} {
		t.Run(source, func(t *testing.T) {
			strictRoutingState(t)
			defer func() {
				if recover() == nil || len(Diagnostics()) == 0 {
					t.Fatal("invalid canonical signature escaped strict refusal")
				}
			}()
			renderGoFileFromJava(t, source)
		})
	}
}
func TestThreadInterruptSourceDeclarationOwnership(t *testing.T) {
	source := `class Thread{int interrupt(){return 73;}}class Probe{Thread t;int run(){return t.interrupt();}}`
	generated := renderGoFileFromJava(t, source)
	if strings.Contains(generated, "ThreadInterruptExecution") || strings.Contains(generated, "stdjava.Thread") {
		t.Fatal("source borrowed native service", generated)
	}
}

// Invalid Java array invocation reaches a separate existing frontend diagnostic
// frontier. Keep it selected and visibly failing until that owner repairs it;
// it must never borrow the native Thread emitter or result metadata.
func TestThreadInterruptInvalidArrayStrictFrontierHeld(t *testing.T) {
	strictRoutingState(t)
	defer func() {
		if recover() == nil || len(Diagnostics()) == 0 {
			t.Fatal("known held: invalid array method escaped strict diagnostic")
		}
	}()
	renderGoFileFromJava(t, `class Probe{Thread[] t;void run(){t.interrupt();}}`)
}

// The original JDK override control is retained as a complete failing gate.
// Native slot tests do not establish source callback wiring or native upcasts.
func TestThreadInterruptSourceOverrideWholeProgramHeldJVM(t *testing.T) {
	threadInterruptFrozenFixture(t, "source-override-held")
}

func TestThreadInterruptOwnersWholeProgramJVM(t *testing.T) {
	threadInterruptFrozenFixture(t, "owners")
}
