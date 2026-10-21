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

func TestJavaNoSuchAlgorithmExceptionReferenceConstructorDispatch(t *testing.T) {
	for _, test := range []struct{ name, prefix, declaration, helper string }{
		{"empty", "", "static java.security.NoSuchAlgorithmException run(){return new java.security.NoSuchAlgorithmException();}", "Message(nil)"},
		{"message", "import java.security.NoSuchAlgorithmException;", "static NoSuchAlgorithmException run(String value){return new NoSuchAlgorithmException(value);}", "Message(value)"},
		{"null-string", "import java.security.*;", "static NoSuchAlgorithmException run(){return new NoSuchAlgorithmException((String)null);}", "Message((*stdjava.JavaString)(nil))"},
		{"cause", "", "static java.security.NoSuchAlgorithmException run(Throwable value){return new java.security.NoSuchAlgorithmException(value);}", "CauseExecution("},
		{"null-cause", "", "static java.security.NoSuchAlgorithmException run(){return new java.security.NoSuchAlgorithmException((Throwable)null);}", "CauseExecution("},
		{"pair", "", "static java.security.NoSuchAlgorithmException run(String text,Throwable cause){return new java.security.NoSuchAlgorithmException(text,cause);}", "MessageCause("},
		{"null-pair", "", "static java.security.NoSuchAlgorithmException run(){return new java.security.NoSuchAlgorithmException(null,null);}", "MessageCause("},
		{"bound-message", "", "static <S extends java.lang.String> java.security.NoSuchAlgorithmException run(S value){return new java.security.NoSuchAlgorithmException(value);}", "Message("},
		{"bound-cause", "", "static <T extends java.lang.Throwable> java.security.NoSuchAlgorithmException run(T value){return new java.security.NoSuchAlgorithmException(value);}", "CauseExecution("},
	} {
		t.Run(test.name, func(t *testing.T) {
			out := renderGoFileFromJava(t, test.prefix+"class Probe{"+test.declaration+"}")
			if !strings.Contains(out, "stdjava.NewJavaNoSuchAlgorithmException"+test.helper) || strings.Contains(out, "stdjava.NewNoSuchAlgorithmException(") {
				t.Fatalf("canonical constructor selected wrong static overload:\n%s", out)
			}
		})
	}
}

func TestJavaNoSuchAlgorithmExceptionCanonicalOwnerAndArgumentShadows(t *testing.T) {
	for _, source := range []string{
		`class NoSuchAlgorithmException {NoSuchAlgorithmException(String s){}}class Probe{static NoSuchAlgorithmException run(String s){return new NoSuchAlgorithmException(s);}}`,
		`import foreign.NoSuchAlgorithmException;class Probe{static NoSuchAlgorithmException run(String s){return new NoSuchAlgorithmException(s);}}`,
		`class Probe<NoSuchAlgorithmException>{NoSuchAlgorithmException value;NoSuchAlgorithmException run(){return value;}}`,
	} {
		out := renderGoFileFromJava(t, source)
		if strings.Contains(out, "NewJavaNoSuchAlgorithmException") {
			t.Fatalf("source/foreign/binder owner borrowed canonical constructor:\n%s", out)
		}
	}
	out := renderGoFileFromJava(t, `class NoSuchAlgorithmException{}class Probe{static java.security.NoSuchAlgorithmException run(String s){return new java.security.NoSuchAlgorithmException(s);}}`)
	if !strings.Contains(out, "stdjava.NewJavaNoSuchAlgorithmExceptionMessage(") {
		t.Fatalf("fully qualified canonical owner lost to source shadow:\n%s", out)
	}
}

func TestJavaNoSuchAlgorithmExceptionInvalidOverloadsStrictlyRefused(t *testing.T) {
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	for index, source := range []string{
		`class Probe{static Object run(){return new java.security.NoSuchAlgorithmException(null);}}`,
		`class Probe{static Object run(){return new java.security.NoSuchAlgorithmException((null));}}`,
		`class Probe{static Object run(){return new java.security.NoSuchAlgorithmException(7);}}`,
		`class Probe{static Object run(Object value){return new java.security.NoSuchAlgorithmException(value);}}`,
		`class Probe{static Object run(Throwable[] value){return new java.security.NoSuchAlgorithmException(value);}}`,
		`class Probe{static Object run(Throwable cause){return new java.security.NoSuchAlgorithmException(7,cause);}}`,
		`class Probe{static Object run(Object cause){return new java.security.NoSuchAlgorithmException("message",cause);}}`,
		`class Probe{static Object run(){return new java.security.NoSuchAlgorithmException("message",null,null);}}`,
		`class String{}class Probe{static Object run(String message){return new java.security.NoSuchAlgorithmException(message);}}`,
		`class Probe<String>{Object run(String message){return new java.security.NoSuchAlgorithmException(message);}}`,
		`class Throwable{}class Probe{static Object run(Throwable cause){return new java.security.NoSuchAlgorithmException(cause);}}`,
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
					t.Fatalf("invalid NSA overload escaped strict refusal: %s", source)
				}
			}()
			renderGoFileFromJava(t, source)
		})
	}
}
