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

func TestJavaGeneralSecurityExceptionReferenceConstructorDispatch(t *testing.T) {
	for _, test := range []struct{ name, prefix, declaration, helper string }{
		{"empty", "", "static java.security.GeneralSecurityException run(){return new java.security.GeneralSecurityException();}", "Message(nil)"},
		{"message", "import java.security.GeneralSecurityException;", "static GeneralSecurityException run(String value){return new GeneralSecurityException(value);}", "Message(value)"},
		{"null-string", "import java.security.*;", "static GeneralSecurityException run(){return new GeneralSecurityException((String)null);}", "Message((*stdjava.JavaString)(nil))"},
		{"cause", "", "static java.security.GeneralSecurityException run(Throwable value){return new java.security.GeneralSecurityException(value);}", "CauseExecution("},
		{"null-cause", "", "static java.security.GeneralSecurityException run(){return new java.security.GeneralSecurityException((Throwable)null);}", "CauseExecution("},
		{"pair", "", "static java.security.GeneralSecurityException run(String text,Throwable cause){return new java.security.GeneralSecurityException(text,cause);}", "MessageCause("},
		{"null-pair", "", "static java.security.GeneralSecurityException run(){return new java.security.GeneralSecurityException(null,null);}", "MessageCause("},
		{"bound-message", "", "static <S extends java.lang.String> java.security.GeneralSecurityException run(S value){return new java.security.GeneralSecurityException(value);}", "Message("},
		{"bound-cause", "", "static <T extends java.lang.Throwable> java.security.GeneralSecurityException run(T value){return new java.security.GeneralSecurityException(value);}", "CauseExecution("},
	} {
		t.Run(test.name, func(t *testing.T) {
			out := renderGoFileFromJava(t, test.prefix+"class Probe{"+test.declaration+"}")
			if !strings.Contains(out, "stdjava.NewJavaGeneralSecurityException"+test.helper) || strings.Contains(out, "stdjava.NewGeneralSecurityException(") {
				t.Fatalf("canonical constructor selected wrong static overload:\n%s", out)
			}
		})
	}
}

func TestJavaGeneralSecurityExceptionCanonicalOwnerAndArgumentShadows(t *testing.T) {
	for _, source := range []string{
		`class GeneralSecurityException {GeneralSecurityException(String s){}}class Probe{static GeneralSecurityException run(String s){return new GeneralSecurityException(s);}}`,
		`import foreign.GeneralSecurityException;class Probe{static GeneralSecurityException run(String s){return new GeneralSecurityException(s);}}`,
		`class Probe<GeneralSecurityException>{GeneralSecurityException value;GeneralSecurityException run(){return value;}}`,
	} {
		out := renderGoFileFromJava(t, source)
		if strings.Contains(out, "NewJavaGeneralSecurityException") {
			t.Fatalf("source/foreign/binder owner borrowed canonical constructor:\n%s", out)
		}
	}
	out := renderGoFileFromJava(t, `class GeneralSecurityException{}class Probe{static java.security.GeneralSecurityException run(String s){return new java.security.GeneralSecurityException(s);}}`)
	if !strings.Contains(out, "stdjava.NewJavaGeneralSecurityExceptionMessage(") {
		t.Fatalf("fully qualified canonical owner lost to source shadow:\n%s", out)
	}
}

func TestJavaGeneralSecurityExceptionInvalidOverloadsStrictlyRefused(t *testing.T) {
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	for index, source := range []string{
		`class Probe{static Object run(){return new java.security.GeneralSecurityException(null);}}`,
		`class Probe{static Object run(){return new java.security.GeneralSecurityException((null));}}`,
		`class Probe{static Object run(){return new java.security.GeneralSecurityException(7);}}`,
		`class Probe{static Object run(Object value){return new java.security.GeneralSecurityException(value);}}`,
		`class Probe{static Object run(Throwable[] value){return new java.security.GeneralSecurityException(value);}}`,
		`class Probe{static Object run(Throwable cause){return new java.security.GeneralSecurityException(7,cause);}}`,
		`class Probe{static Object run(Object cause){return new java.security.GeneralSecurityException("message",cause);}}`,
		`class Probe{static Object run(){return new java.security.GeneralSecurityException("message",null,null);}}`,
		`class String{}class Probe{static Object run(String message){return new java.security.GeneralSecurityException(message);}}`,
		`class Probe<String>{Object run(String message){return new java.security.GeneralSecurityException(message);}}`,
		`class Throwable{}class Probe{static Object run(Throwable cause){return new java.security.GeneralSecurityException(cause);}}`,
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
					t.Fatalf("invalid GSE overload escaped strict refusal: %s", source)
				}
			}()
			renderGoFileFromJava(t, source)
		})
	}
}
