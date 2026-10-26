package transpiler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Retain the compiler's original output even when the strict behavior build fails.
func TestVolatileCommentAssignmentFrozenEmission(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "volatile_comment_assignment", "VolatileCommentAssignmentProbe.java"))
	if err != nil {
		t.Fatal(err)
	}
	generated := renderGoFileFromJava(t, string(source))
	dir := os.Getenv("JAVA2GO_COMMENT_ASSIGNMENT_EMISSION_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	file, err := os.OpenFile(filepath.Join(dir, "generated.go"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.WriteString(generated); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			t.Errorf("close failed emission file: %v", closeErr)
		}
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
}

// The oracle is actual captured standalone JDK21 stdout for the byteexact peer
// fixture. No expected result is synthesized and no Java expectation is changed.
func TestVolatileCommentAssignmentJDK21(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "volatile_comment_assignment", "VolatileCommentAssignmentProbe.java"))
	if err != nil {
		t.Fatal(err)
	}
	oracle := os.Getenv("JAVA2GO_COMMENT_ASSIGNMENT_ORACLE_FILE")
	var want []byte
	if oracle == "" {
		want = volatileCommentAssignmentJavaOracle(t, source)
	} else {
		want, err = os.ReadFile(oracle)
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(want) == 0 {
		t.Fatal("actual JDK oracle is empty")
	}
	t.Logf("JDK21 volatile comment assignment oracle: %q", string(want))
	generated := renderGoFileFromJava(t, string(source))
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("testing";"os";"io")
func TestCommentedVolatileAssignmentMain(t *testing.T){old:=os.Stdout;r,w,err:=os.Pipe();if err!=nil{t.Fatal(err)};os.Stdout=w;defer func(){os.Stdout=old}();Main();if err:=w.Close();err!=nil{t.Fatal(err)};got,err:=io.ReadAll(r);if err!=nil{t.Fatal(err)};if err:=r.Close();err!=nil{t.Fatal(err)};if string(got)!=%q{t.Fatalf("JDK %%q != Go %%q",%q,got)}}`, string(want), string(want)))
}

// Ordinary package tests observe the unchanged standalone fixture directly.
// Sealed runners retain their separately captured actual-oracle file above.
func volatileCommentAssignmentJavaOracle(t *testing.T, source []byte) []byte {
	t.Helper()
	campaignCollectionReferenceJDK21(t)
	javac, java := "javac", "java"
	if home := os.Getenv("JAVA_HOME"); home != "" {
		javac, java = filepath.Join(home, "bin", "javac"), filepath.Join(home, "bin", "java")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "VolatileCommentAssignmentProbe.java")
	if err := os.WriteFile(input, source, 0600); err != nil {
		t.Fatal(err)
	}
	compiled := campaignStringBoundedRun(t, "volatile comment JDK21 compile", dir, javac, 30*time.Second, "--release", "21", "-encoding", "UTF-8", "-proc:none", "-d", dir, input)
	campaignStringRequireSuccess(t, "volatile comment JDK21 compile", compiled)
	if len(compiled.stdout) != 0 || len(compiled.stderr) != 0 {
		t.Fatalf("unexpected JDK21 compilation output: stdout=%q stderr=%q", compiled.stdout, compiled.stderr)
	}
	observed := campaignStringBoundedRun(t, "volatile comment JDK21 execution", dir, java, 30*time.Second, "-cp", dir, "VolatileCommentAssignmentProbe")
	campaignStringRequireSuccess(t, "volatile comment JDK21 execution", observed)
	if len(observed.stderr) != 0 {
		t.Fatalf("unexpected JDK21 oracle stderr: %q", observed.stderr)
	}
	return observed.stdout
}
