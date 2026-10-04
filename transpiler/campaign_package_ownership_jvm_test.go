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

// The generated build uses the same pinned dependency metadata as its runtime.
// Keep module setup outside readonly builds; do not alter generated Go source.
func TestCampaignPackageOwnershipSeparateGraphsJVM(t *testing.T) {
	withCleanDiagnostics(t)
	files := map[string]string{
		"pom.xml":                      `<project><modelVersion>4.0.0</modelVersion><groupId>ownership</groupId><artifactId>probe</artifactId><version>1</version></project>`,
		"src/main/java/p/Value.java":   `package p;public class Value{public static int answer(){return 19;}public static class Member{public static int answer(){return 17;}}}`,
		"src/main/java/q/Value.java":   `package q;public class Value{public static int answer(){return 43;}public static class Member{public static int answer(){return 41;}}}`,
		"src/main/java/p/Base.java":    `package p;public class Base{protected static int inherited(){return 73;}}`,
		"src/main/java/app/Child.java": `package app;public class Child extends p.Base{public static int read(){return inherited();}}`,
		"src/main/java/app/Main.java":  `package app;public class Main{public static void main(String[] args){System.out.println(p.Value.answer());System.out.println(p.Value.Member.answer());System.out.println(q.Value.answer());System.out.println(q.Value.Member.answer());System.out.println(Child.read());}}`,
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
	if string(want) != "19\n17\n43\n41\n73\n" {
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
