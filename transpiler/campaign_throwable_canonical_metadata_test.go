package transpiler

import (
	"bytes"
	"context"
	"github.com/NickyBoy89/java2go/symbol"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func TestCampaignBuiltinThrowableCanonicalClassJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>throwable-metadata</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
public class Main {
 static String describe(Throwable value,Class<?> expected){
  return value.getClass().getName()+":"+value.getClass().getSimpleName()+":"+(value.getClass()==expected);
 }
 public static void main(String[] args) throws Exception {
  System.out.println(describe(new NullPointerException("null"),NullPointerException.class));
  System.out.println(describe(new java.io.IOException("io"),java.io.IOException.class));
  System.out.println(describe(new java.util.ConcurrentModificationException("modified"),java.util.ConcurrentModificationException.class));
  System.out.println(describe(new java.util.concurrent.CancellationException("cancel"),java.util.concurrent.CancellationException.class));
  System.out.println(describe(new java.text.ParseException("parse",3),java.text.ParseException.class));
  System.out.println(describe(new java.security.NoSuchAlgorithmException("algorithm"),java.security.NoSuchAlgorithmException.class));
  System.out.println(Class.forName("java.lang.NullPointerException")==NullPointerException.class);
  System.out.println(NullPointerException.class.getSuperclass()==RuntimeException.class);
  System.out.println(Throwable.class.isAssignableFrom(java.io.IOException.class));
  System.out.println(Exception.class.isAssignableFrom(NullPointerException.class));
  System.out.println(RuntimeException.class.isAssignableFrom(java.io.IOException.class));
 }
}`}, "probe.Main", "java.lang.NullPointerException:NullPointerException:true\njava.io.IOException:IOException:true\njava.util.ConcurrentModificationException:ConcurrentModificationException:true\njava.util.concurrent.CancellationException:CancellationException:true\njava.text.ParseException:ParseException:true\njava.security.NoSuchAlgorithmException:NoSuchAlgorithmException:true\ntrue\ntrue\ntrue\ntrue\nfalse\n")
}

func TestCampaignThrowableCanonicalSourceShadowAndArraysJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>throwable-shadow</artifactId><version>1</version></project>`,
		"src/main/java/shadow/NullPointerException.java": `package shadow;
public class NullPointerException extends RuntimeException {
 public NullPointerException(String message){super(message);}
 @Override public String toString(){return "source-cause:"+getMessage();}
}`,
		"src/main/java/probe/Main.java": `package probe;
public class Main {
 static class NestedFailure extends shadow.NullPointerException {NestedFailure(){super("nested");}}
 public static void main(String[] args){
  Throwable source=new shadow.NullPointerException("source");
  NestedFailure nested=new NestedFailure();
  Throwable builtin=new java.lang.NullPointerException("builtin");
  System.out.println(source.getClass().getName()+":"+nested.getClass().getName());
  System.out.println((source.getClass()==shadow.NullPointerException.class)+":"+(source.getClass()==builtin.getClass()));
  System.out.println(RuntimeException.class.isAssignableFrom(source.getClass())+":"+Throwable.class.isAssignableFrom(nested.getClass()));
  Throwable[] failures=new Throwable[]{source,nested,builtin,new java.io.IOException("io")};
  System.out.println((failures[0]==source)+":"+(failures[1]==nested)+":"+(failures[2].getClass()==java.lang.NullPointerException.class));
  Object[] runtime=new RuntimeException[1];runtime[0]=source;
  try {runtime[0]=new java.io.IOException("checked");System.out.println("bad-store");}
  catch(ArrayStoreException expected){System.out.println("store-checked:"+(runtime[0]==source));}
  try {throw nested;}
  catch(shadow.NullPointerException caught){System.out.println("source-caught:"+(caught==nested));}
  try {throw source;}
  catch(RuntimeException caught){System.out.println("runtime-caught:"+(caught==source));}
  catch(Throwable wrong){System.out.println("bad-catch");}
  Exception causeOnly=new Exception(source);
  System.out.println(causeOnly.getMessage()+":"+(causeOnly.getCause()==source));
  java.io.IOException io=new java.io.IOException("io");
  Exception builtinCause=new Exception(io);
  System.out.println(builtinCause.getMessage()+":"+(builtinCause.getCause()==io));
  Exception empty=new Exception("");Exception missing=new Exception((String)null);
  System.out.println(empty.getMessage().length()+":"+(missing.getMessage()==null));
 }
}`}, "probe.Main", "shadow.NullPointerException:probe.Main$NestedFailure\ntrue:false\ntrue:true\ntrue:true:true\nstore-checked:true\nsource-caught:true\nruntime-caught:true\nsource-cause:source:true\njava.io.IOException: io:true\n0:true\n")
}

func TestCampaignThrowableUnnamedPackageShadowJVM(t *testing.T) {
	const source = `class NullPointerException extends RuntimeException {}
public class Main {public static void main(String[] args){
 Throwable source=new NullPointerException();Throwable builtin=new java.lang.NullPointerException("builtin");
 System.out.println(source.getClass().getName()+":"+builtin.getClass().getName());
 System.out.println((source.getClass()==NullPointerException.class)+":"+(source.getClass()==java.lang.NullPointerException.class));
 System.out.println(NullPointerException.class.isAssignableFrom(builtin.getClass())+":"+RuntimeException.class.isAssignableFrom(source.getClass()));
}}`
	runCampaignThrowableStrictSingleFileOracle(t, source, "NullPointerException:java.lang.NullPointerException\ntrue:false\nfalse:true\n")
}

func runCampaignThrowableStrictSingleFileOracle(t *testing.T, source, expected string, additional ...map[string]string) {
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
		cmd.Env = append(os.Environ(), "GOWORK=off", "JAVA2GO_ASSERTIONS=false")
		cmd.WaitDelay = 5 * time.Second
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		t.Logf("%s: %v", stage, cmd.Args)
		if ctx.Err() != nil || err != nil {
			t.Fatalf("%s failed: %v timeout=%v\nstdout: %s\nstderr: %s", stage, err, ctx.Err(), stdout.Bytes(), stderr.Bytes())
		}
		return stdout.Bytes(), stderr.Bytes()
	}
	root := t.TempDir()
	input := filepath.Join(root, "source")
	path := filepath.Join(input, "Main.java")
	if err := writeProjectFile(path, []byte(source)); err != nil {
		t.Fatal(err)
	}
	sources := []string{path}
	for _, files := range additional {
		for name, content := range files {
			extra := filepath.Join(input, name)
			if err := writeProjectFile(extra, []byte(content)); err != nil {
				t.Fatal(err)
			}
			sources = append(sources, extra)
		}
	}
	sort.Strings(sources)
	classes := filepath.Join(root, "classes")
	run("javac", root, 5*time.Minute, javac, append([]string{"--release", "21", "-encoding", "UTF-8", "-d", classes}, sources...)...)
	want, wantErr := run("JVM", root, time.Minute, java, "-cp", classes, "Main")
	if string(want) != expected || len(wantErr) != 0 {
		t.Fatalf("invalid JVM oracle: stdout=%q stderr=%q expected=%q", want, wantErr, expected)
	}
	t.Logf("JVM oracle: %q", want)
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	compiler := filepath.Join(root, "java2go")
	run("compiler build", repo, 5*time.Minute, "go", "build", "-o", compiler, "./cmd/java2go")
	generated := filepath.Join(root, "generated")
	run("strict single-file transpilation", repo, 5*time.Minute, compiler, "-strict", "-w", "-output", generated, input)
	// Query the same production resolver used by the CLI, not a guessed suffix.
	previousGlobal := symbol.GlobalScope
	previousAnnotations := excludedAnnotations
	diagnostics.mu.Lock()
	previousStrict, previousDiagnostics := diagnostics.strict, append([]Diagnostic(nil), diagnostics.items...)
	diagnostics.mu.Unlock()
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	defer func() {
		symbol.GlobalScope = previousGlobal
		excludedAnnotations = previousAnnotations
		diagnostics.mu.Lock()
		diagnostics.strict, diagnostics.items = previousStrict, previousDiagnostics
		diagnostics.mu.Unlock()
	}()
	if err := runInternal([]string{"-strict", "-q", "-sync", input}, io.Discard, false); err != nil {
		t.Fatal(err)
	}
	entry := ""
	if pkg := symbol.GlobalScope.FindPackage(""); pkg != nil {
		for _, file := range pkg.Files {
			for _, class := range file.TopLevelClasses {
				if class.Class.OriginalName == "Main" {
					for _, method := range class.Methods {
						if projectMain(method) {
							entry = method.Name
						}
					}
				}
			}
		}
	}
	if entry == "" {
		t.Fatal("resolved Java main entry not found")
	}
	t.Logf("resolved entry: %s", entry)
	// The only handwritten driver statement invokes the resolved Java entry.
	if err := writeProjectFile(filepath.Join(generated, "launcher.go"), []byte("package main\nfunc main(){"+entry+"()}\n")); err != nil {
		t.Fatal(err)
	}
	mod := "module generated\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\nreplace github.com/NickyBoy89/java2go => " + repo + "\n"
	if err := writeProjectFile(filepath.Join(generated, "go.mod"), []byte(mod)); err != nil {
		t.Fatal(err)
	}
	run("all generated packages race build", generated, 5*time.Minute, "go", "build", "-race", "-mod=mod", "./...")
	binary := filepath.Join(root, "app")
	run("entry point race build", generated, 5*time.Minute, "go", "build", "-race", "-mod=mod", "-o", binary, ".")
	got, gotErr := run("generated Go", root, time.Minute, binary)
	if !bytes.Equal(got, want) || !bytes.Equal(gotErr, wantErr) {
		t.Fatalf("Go stdout=%q stderr=%q differs from JVM stdout=%q stderr=%q", got, gotErr, want, wantErr)
	}
}
