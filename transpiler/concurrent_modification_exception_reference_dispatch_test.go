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

func TestJavaConcurrentModificationExceptionReferenceConstructorDispatch(t *testing.T) {
	for _, test := range []struct{ name, prefix, declaration, helper string }{
		{"empty", "", "static java.util.ConcurrentModificationException run(){return new java.util.ConcurrentModificationException();}", "Message(nil)"},
		{"message", "import java.util.ConcurrentModificationException;", "static ConcurrentModificationException run(String value){return new ConcurrentModificationException(value);}", "Message(value)"},
		{"null-string", "import java.util.*;", "static ConcurrentModificationException run(){return new ConcurrentModificationException((String)null);}", "Message((*stdjava.JavaString)(nil))"},
		{"cause", "", "static java.util.ConcurrentModificationException run(Throwable value){return new java.util.ConcurrentModificationException(value);}", "CauseExecution("},
		{"null-cause", "", "static java.util.ConcurrentModificationException run(){return new java.util.ConcurrentModificationException((Throwable)null);}", "CauseExecution("},
		{"pair", "", "static java.util.ConcurrentModificationException run(String text,Throwable cause){return new java.util.ConcurrentModificationException(text,cause);}", "MessageCause("},
		{"null-pair", "", "static java.util.ConcurrentModificationException run(){return new java.util.ConcurrentModificationException(null,null);}", "MessageCause("},
		{"bound-message", "", "static <S extends java.lang.String> java.util.ConcurrentModificationException run(S value){return new java.util.ConcurrentModificationException(value);}", "Message("},
		{"bound-cause", "", "static <T extends java.lang.Throwable> java.util.ConcurrentModificationException run(T value){return new java.util.ConcurrentModificationException(value);}", "CauseExecution("},
	} {
		t.Run(test.name, func(t *testing.T) {
			out := renderGoFileFromJava(t, test.prefix+"class Probe{"+test.declaration+"}")
			if !strings.Contains(out, "stdjava.NewJavaConcurrentModificationException"+test.helper) || strings.Contains(out, "stdjava.NewConcurrentModificationException(") {
				t.Fatalf("canonical constructor selected wrong static overload:\n%s", out)
			}
		})
	}
}

func TestJavaConcurrentModificationExceptionCanonicalOwnerAndArgumentShadows(t *testing.T) {
	for _, source := range []string{
		`class ConcurrentModificationException {ConcurrentModificationException(String s){}}class Probe{static ConcurrentModificationException run(String s){return new ConcurrentModificationException(s);}}`,
		`import foreign.ConcurrentModificationException;class Probe{static ConcurrentModificationException run(String s){return new ConcurrentModificationException(s);}}`,
		`class Probe<ConcurrentModificationException>{ConcurrentModificationException value;ConcurrentModificationException run(){return value;}}`,
	} {
		out := renderGoFileFromJava(t, source)
		if strings.Contains(out, "NewJavaConcurrentModificationException") {
			t.Fatalf("source/foreign/binder owner borrowed canonical constructor:\n%s", out)
		}
	}
	out := renderGoFileFromJava(t, `class ConcurrentModificationException{}class Probe{static java.util.ConcurrentModificationException run(String s){return new java.util.ConcurrentModificationException(s);}}`)
	if !strings.Contains(out, "stdjava.NewJavaConcurrentModificationExceptionMessage(") {
		t.Fatalf("fully qualified canonical owner lost to source shadow:\n%s", out)
	}
}

func TestJavaConcurrentModificationExceptionInvalidOverloadsStrictlyRefused(t *testing.T) {
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	for index, source := range []string{
		`class Probe{static Object run(){return new java.util.ConcurrentModificationException(null);}}`,
		`class Probe{static Object run(){return new java.util.ConcurrentModificationException((null));}}`,
		`class Probe{static Object run(){return new java.util.ConcurrentModificationException(7);}}`,
		`class Probe{static Object run(Object value){return new java.util.ConcurrentModificationException(value);}}`,
		`class Probe{static Object run(Throwable[] value){return new java.util.ConcurrentModificationException(value);}}`,
		`class Probe{static Object run(Throwable cause){return new java.util.ConcurrentModificationException(7,cause);}}`,
		`class Probe{static Object run(Object cause){return new java.util.ConcurrentModificationException("message",cause);}}`,
		`class Probe{static Object run(){return new java.util.ConcurrentModificationException("message",null,null);}}`,
		`class String{}class Probe{static Object run(String message){return new java.util.ConcurrentModificationException(message);}}`,
		`class Probe<String>{Object run(String message){return new java.util.ConcurrentModificationException(message);}}`,
		`class Throwable{}class Probe{static Object run(Throwable cause){return new java.util.ConcurrentModificationException(cause);}}`,
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
					t.Fatalf("invalid CME overload escaped strict refusal: %s", source)
				}
			}()
			renderGoFileFromJava(t, source)
		})
	}
}
