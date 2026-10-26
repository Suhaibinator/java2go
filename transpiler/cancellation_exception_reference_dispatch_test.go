package transpiler

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJavaCancellationExceptionReferenceConstructorDispatch(t *testing.T) {
	for _, test := range []struct{ name, prefix, declaration, helper string }{
		{"empty", "", "static java.util.concurrent.CancellationException run(){return new java.util.concurrent.CancellationException();}", "Message(nil)"},
		{"message", "import java.util.concurrent.CancellationException;", "static CancellationException run(String value){return new CancellationException(value);}", "Message("},
		{"bare-null", "", "static Object run(){return new java.util.concurrent.CancellationException(null);}", "Message("},
		{"bound-message", "", "static <S extends String> Object run(S value){return new java.util.concurrent.CancellationException(value);}", "Message("},
	} {
		t.Run(test.name, func(t *testing.T) {
			out := renderGoFileFromJava(t, test.prefix+"class Probe{"+test.declaration+"}")
			if !strings.Contains(out, "stdjava.NewJavaCancellationException"+test.helper) || strings.Contains(out, "stdjava.NewCancellationException(") {
				t.Fatalf("canonical constructor selected wrong static overload:\n%s", out)
			}
		})
	}
}

func TestJavaCancellationExceptionCanonicalOwnerAndArgumentShadows(t *testing.T) {
	for _, source := range []string{
		`class CancellationException {CancellationException(String s){}}class Probe{static CancellationException run(String s){return new CancellationException(s);}}`,
		`import foreign.CancellationException;class Probe{static CancellationException run(String s){return new CancellationException(s);}}`,
		`class Probe<CancellationException>{CancellationException value;CancellationException run(){return value;}}`,
	} {
		out := renderGoFileFromJava(t, source)
		if strings.Contains(out, "NewJavaCancellationException") {
			t.Fatalf("source/foreign/binder owner borrowed canonical constructor:\n%s", out)
		}
	}
	out := renderGoFileFromJava(t, `class CancellationException{}class Probe{static java.util.concurrent.CancellationException run(String s){return new java.util.concurrent.CancellationException(s);}}`)
	if !strings.Contains(out, "stdjava.NewJavaCancellationExceptionMessage(") {
		t.Fatalf("fully qualified canonical owner lost to source shadow:\n%s", out)
	}
}

func TestJavaCancellationExceptionInvalidOverloadsStrictlyRefused(t *testing.T) {
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	for index, source := range []string{
		`class Probe{static Object run(Throwable value){return new java.util.concurrent.CancellationException(value);}}`,
		`class Probe{static Object run(){return new java.util.concurrent.CancellationException("message",null);}}`,
		`class Probe{static Object run(){return new java.util.concurrent.CancellationException(7);}}`,
		`class Probe{static Object run(Object value){return new java.util.concurrent.CancellationException(value);}}`,
		`class Probe{static Object run(String[] value){return new java.util.concurrent.CancellationException(value);}}`,
		`class String{}class Probe{static Object run(String value){return new java.util.concurrent.CancellationException(value);}}`,
		`class Probe<String>{Object run(String value){return new java.util.concurrent.CancellationException(value);}}`,
	} {
		t.Run(string(rune('a'+index)), func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "Probe.java")
			if err := os.WriteFile(path, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, javac, "--release", "21", "-encoding", "UTF-8", "-d", filepath.Join(root, "classes"), path)
			output, err := cmd.CombinedOutput()
			t.Logf("negative javac: %v error=%v output=%s", cmd.Args, err, output)
			if ctx.Err() != nil || err == nil || len(output) == 0 {
				t.Fatalf("JDK failed to reject invalid constructor: err=%v timeout=%v output=%s", err, ctx.Err(), output)
			}
			strictRoutingState(t)
			defer func() {
				if failure := recover(); failure == nil || len(Diagnostics()) == 0 {
					t.Fatalf("invalid Cancellation overload escaped strict refusal: %s", source)
				}
			}()
			renderGoFileFromJava(t, source)
		})
	}
}
