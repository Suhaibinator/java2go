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

func TestReadonlyHeapWholeProgramJVM(t *testing.T)         { readonlyFrozenFixture(t, "values") }
func TestReadonlyReadConsumerWholeProgramJVM(t *testing.T) { readonlyFrozenFixture(t, "read") }
func TestReadonlySourceAndBinderWholeProgramJVM(t *testing.T) {
	readonlyFrozenFixture(t, "binder-held")
}
func TestReadonlyForeignWholeProgramJVM(t *testing.T) { readonlyFrozenFixture(t, "foreign-held") }
func readonlyFrozenFixture(t *testing.T, name string) {
	t.Helper()
	fixture := filepath.Join("..", "campaign", "reproducers", "jdk-bytebuffer-readonly", name)
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
func TestReadonlyIntrinsicInvalidSignatureStrictRefusal(t *testing.T) {
	for _, source := range []string{
		`import java.nio.ByteBuffer;class Probe{static Object run(ByteBuffer b){return b.asReadOnlyBuffer(1);}}`,
		`import java.nio.ByteBuffer;class Probe{static Object run(ByteBuffer b){return b.put(null);}}`,
		`import java.nio.ByteBuffer;class Probe{static Object run(ByteBuffer b){return b.put(0,null,0,0);}}`,
		`import java.nio.ByteBuffer;class Probe{static int run(ByteBuffer b){return b.getChar(0,0);}}`,
		`import java.nio.ByteBuffer;class Probe{static Object run(ByteBuffer b){return b.putChar(0L,'x');}}`,
		`import java.nio.ByteBuffer;class Probe{static Object run(ByteBuffer b){return b.put(0,new int[1]);}}`,
		`import java.nio.channels.FileChannel;class Probe{static int run(FileChannel c){return c.read(1);}}`,
		`import java.nio.channels.FileChannel;class Probe{static int run(FileChannel c){return c.read((java.nio.ByteBuffer[])null);}}`,
	} {
		t.Run(source, func(t *testing.T) {
			strictRoutingState(t)
			defer func() {
				if recover() == nil || len(Diagnostics()) == 0 {
					t.Fatal("unsupported signature escaped strict refusal")
				}
			}()
			renderGoFileFromJava(t, source)
		})
	}
}
func TestReadonlyIntrinsicSourceDeclarationOwnership(t *testing.T) {
	for _, source := range []string{
		`class ByteBuffer{String asReadOnlyBuffer(){return "source";}int isReadOnly(){return 3;}}class Probe{static String run(ByteBuffer b){return b.asReadOnlyBuffer();}}`,
		`class FileChannel{String read(String s){return s;}}class Probe{static String run(FileChannel c){return c.read("source");}}`,
	} {
		generated := renderGoFileFromJava(t, source)
		for _, forbidden := range []string{"FileChannelReadExecution", "stdjava.ByteBuffer"} {
			if strings.Contains(generated, forbidden) {
				t.Fatalf("source declaration borrowed %s: %s", forbidden, generated)
			}
		}
	}
}

func TestReadonlyMutableErrorsWholeProgramJVM(t *testing.T) {
	readonlyFrozenFixture(t, "mutable-errors")
}

func TestReadonlyOverflowOwnerWholeProgramJVM(t *testing.T) {
	readonlyFrozenFixture(t, "overflow-owner")
}

func TestReadonlyEmitterResultApplicability(t *testing.T) {
	for _, c := range []struct {
		source, result string
		want           bool
	}{
		{`import java.nio.ByteBuffer;class Probe{ByteBuffer b;Object run(){return b.asReadOnlyBuffer();}}`, "java.nio.ByteBuffer", true},
		{`import java.nio.ByteBuffer;class Probe{ByteBuffer b;boolean run(){return b.isReadOnly();}}`, "boolean", true},
		{`import java.nio.ByteBuffer;class Probe{ByteBuffer b;char run(){return b.getChar(0);}}`, "char", true},
		{`import java.nio.channels.FileChannel;import java.nio.ByteBuffer;class Probe{FileChannel c;ByteBuffer b;int run()throws Exception{return c.read(b);}}`, "int", true},
		{`import java.nio.ByteBuffer;class Probe{ByteBuffer b;Object run(){return b.asReadOnlyBuffer(1);}}`, "", false},
		{`import java.nio.ByteBuffer;class Probe{ByteBuffer b;char run(){return b.getChar(0,0);}}`, "", false},
		{`import java.nio.ByteBuffer;class Probe{ByteBuffer b;Object run(){return b.put(0,null,0,0);}}`, "", false},
		{`import java.nio.channels.FileChannel;class Probe{FileChannel c;int run()throws Exception{return c.read(7);}}`, "", false},
		{`import java.nio.channels.FileChannel;class Probe{FileChannel c;int run()throws Exception{return c.read((java.nio.ByteBuffer[])null);}}`, "", false},
		{`class FileChannel{String read(int i){return "source";}}class Probe{FileChannel c;String run(){return c.read(7);}}`, "", false},
		{`import foreign.ByteBuffer;class Probe{ByteBuffer b;Object run(){return b.asReadOnlyBuffer();}}`, "", false},
		{`class Probe<ByteBuffer extends java.nio.ByteBuffer>{ByteBuffer b;boolean run(){return b.isReadOnly();}}`, "", false},
	} {
		t.Run(c.source, func(t *testing.T) {
			helper := setupParseHelper(t, c.source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			result, ok := inferIntrinsicMethodResultType(invocation, ctx, helper.File.Source)
			if ok != c.want || ok && result != c.result {
				t.Fatalf("result applicability %q/%t want %q/%t", result, ok, c.result, c.want)
			}
		})
	}
}
