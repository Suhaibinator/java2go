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

// Prepared, unexecuted import control: the JDK must validate these observations
// before generated Go parity is considered. Lexical declarations, ordinary and
// static demand imports, and cross-file hierarchy contexts all resolve afresh.
// Build metadata is prepared before readonly builds; generated Go is untouched.
func TestCampaignImportSyntaxStrictJVM(t *testing.T) {
	withCleanDiagnostics(t)
	files := map[string]string{
		"pom.xml":                       `<project><modelVersion>4.0.0</modelVersion><groupId>importfacts</groupId><artifactId>probe</artifactId><version>1</version></project>`,
		"src/main/java/p/Owner.java":    `package p;public class Owner{public static int answer(){return 11;}public static class Box{public static int answer(){return 17;}}}`,
		"src/main/java/p/Value.java":    `package p;public class Value{public static int answer(){return 19;}}`,
		"src/main/java/q/Value.java":    `package q;public class Value{public static int answer(){return 43;}}`,
		"src/main/java/q/Other.java":    `package q;public class Other{public static int second(){return 13;}}`,
		"src/main/java/app/Helper.java": `package app;import p.*;import static p.Owner.*;public class Helper{public static int read(){return answer()+Box.answer()+Value.answer();}}`,
		"src/main/java/app/Main.java":   `package app;import p.*;import static p.Owner.answer;import static q.Other.second;public class Main{public static int answer(){return 71;}public static class Value{public static int answer(){return 53;}}public static void main(String[] args){System.out.println(answer());System.out.println(Value.answer());System.out.println(second());System.out.println(Helper.read());System.out.println(p.Value.answer());System.out.println(q.Value.answer());}}`,
	}
	root := t.TempDir()
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	var sources []string
	for name, source := range files {
		path := filepath.Join(root, name)
		if err := writeProjectFile(path, []byte(source)); err != nil {
			t.Fatal(err)
		}
		if filepath.Ext(name) == ".java" {
			sources = append(sources, path)
		}
	}
	sort.Strings(sources)
	run := func(stage, dir, name string, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-p=1 -mod=readonly")
		cmd.WaitDelay = 2 * time.Second
		var out, errout bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &errout
		if err := cmd.Run(); err != nil || ctx.Err() != nil {
			t.Fatalf("%s: %v / %v stdout=%q stderr=%q", stage, err, ctx.Err(), out.String(), errout.String())
		}
		if errout.Len() != 0 {
			t.Fatalf("%s stderr=%q", stage, errout.String())
		}
		t.Logf("%s: %v passed", stage, cmd.Args)
		return out.Bytes()
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
	run("javac", root, javac, append([]string{"--release", "21", "-d", classes}, sources...)...)
	want := run("JVM", root, java, "-cp", classes, "app.Main")
	if string(want) != "71\n53\n13\n47\n19\n43\n" {
		t.Fatalf("invalid Java observations %q", want)
	}
	generated := filepath.Join(root, "generated")
	// The ordinary project API performs strict conversion of every source file.
	if err := Run([]string{"-strict", "-maven", root, "-main-class", "app.Main", "-runtime", repo, "-output", generated}); err != nil {
		t.Fatal(err)
	}
	originalMod, err := os.ReadFile(filepath.Join(generated, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeMod, err := os.ReadFile(filepath.Join(repo, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeSum, err := os.ReadFile(filepath.Join(repo, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	firstLine := strings.SplitN(string(originalMod), "\n", 2)[0]
	buildMod := firstLine + "\n" + strings.SplitN(string(runtimeMod), "\n", 2)[1] + "\nrequire github.com/NickyBoy89/java2go v0.0.0\nreplace github.com/NickyBoy89/java2go => " + filepath.ToSlash(repo) + "\n"
	if err := os.WriteFile(filepath.Join(generated, "go.mod"), []byte(buildMod), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(generated, "go.sum"), runtimeSum, 0600); err != nil {
		t.Fatal(err)
	}
	run("all generated packages race build", generated, "go", "build", "-p=1", "-mod=readonly", "-race", "./...")
	binary := filepath.Join(root, "generated-app")
	run("entry point race build", generated, "go", "build", "-p=1", "-mod=readonly", "-race", "-o", binary, "./cmd/app")
	got := run("generated Go", root, binary)
	if !bytes.Equal(got, want) {
		t.Fatalf("Go=%q Java=%q", got, want)
	}
	if artifactDir := os.Getenv("JAVA2GO_TEST_ARTIFACT_DIR"); artifactDir != "" {
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			name := entry.Name()
			if filepath.Ext(path) != ".go" && filepath.Ext(path) != ".java" && name != "go.mod" && name != "go.sum" && name != "pom.xml" {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return writeProjectFile(filepath.Join(artifactDir, t.Name(), relative), raw)
		}); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("matching observations %q", got)
}
