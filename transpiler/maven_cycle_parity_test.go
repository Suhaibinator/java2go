package transpiler

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Java packages may form cycles, including classes with the same simple name.
// Run the compiler in a process so this oracle cannot race global symbol state.
func TestMavenPackageCycleJVMParity(t *testing.T) {
	files := map[string]string{
		"pom.xml":                             `<project><groupId>example</groupId><artifactId>cycle</artifactId><version>1</version></project>`,
		"src/main/java/cycle/left/Node.java":  `package cycle.left; public class Node { public int value; public Node(int value) { this.value = value; } public static int result() { return new cycle.right.Node(20).twice() + 2; } }`,
		"src/main/java/cycle/right/Node.java": `package cycle.right; public class Node extends cycle.left.Node { private int value; public Node(int value) { super(value); this.value = value; } public int twice() { cycle.left.Node Node = new cycle.left.Node(value); return Node.value * 2; } }`,
		"src/main/java/cycle/app/Main.java":   `package cycle.app; import cycle.left.Node; public class Main { public static void main(String[] args) { System.out.println(Node.result()); System.out.println(cycle.left.Node.class.getName()); System.out.println(cycle.right.Node.class.getName()); } }`,
	}
	runCampaignCompilerProjectOracle(t, files, "cycle.app.Main", "42\ncycle.left.Node\ncycle.right.Node\n")
}

func runCampaignCompilerProjectOracle(t *testing.T, files map[string]string, mainClass, expected string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatalf("JDK21 javac required: %v", err)
	}
	java, err := campaignCompilerJavaTool("java")
	if err != nil {
		t.Fatalf("JDK21 java required: %v", err)
	}
	root := t.TempDir()
	sources := []string{}
	for name, source := range files {
		path := filepath.Join(root, name)
		if err := writeProjectFile(path, []byte(source)); err != nil {
			t.Fatal(err)
		}
		if filepath.Ext(name) == ".java" {
			sources = append(sources, path)
		}
	}
	classes := filepath.Join(root, "classes")
	if output, err := exec.CommandContext(ctx, javac, append([]string{"--release", "21", "-d", classes}, sources...)...).CombinedOutput(); err != nil {
		t.Fatalf("javac: %v\n%s", err, output)
	}
	want, err := exec.CommandContext(ctx, java, "-cp", classes, mainClass).CombinedOutput()
	if err != nil {
		t.Fatalf("java: %v\n%s", err, want)
	}
	if string(want) != expected {
		t.Fatalf("unexpected JVM oracle %q", want)
	}
	t.Logf("JVM oracle: %q", want)
	repo, _ := filepath.Abs("..")
	generated := filepath.Join(root, "generated")
	compiler := exec.CommandContext(ctx, "go", "run", "./cmd/java2go", "-maven", root, "-main-class", mainClass, "-runtime", repo, "-output", generated)
	compiler.Dir = repo
	if output, err := compiler.CombinedOutput(); err != nil {
		t.Fatalf("compiler: %v\n%s", err, output)
	}
	command := exec.CommandContext(ctx, "go", "run", "-mod=mod", "./cmd/app")
	command.Dir = generated
	command.Env = append(os.Environ(), "GOWORK=off")
	got, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("generated Go: %v\n%s", err, got)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Go %q differs from JVM %q", got, want)
	}
}

func campaignCompilerJavaTool(name string) (string, error) {
	if home := os.Getenv("JAVA_HOME"); home != "" {
		return exec.LookPath(filepath.Join(home, "bin", name))
	}
	return exec.LookPath(name)
}

func TestMavenCycleEntrypointsJVMParity(t *testing.T) {
	files := map[string]string{
		"pom.xml":                               `<project><groupId>example</groupId><artifactId>entries</artifactId><version>1</version></project>`,
		"src/main/java/entries/left/Main.java":  `package entries.left; public class Main { public static int read(){return new entries.right.Main().value();} public static void main(String[] args){System.out.println(read());} }`,
		"src/main/java/entries/right/Main.java": `package entries.right; public class Main { public int value(){return 17;} public entries.left.Main other(){return new entries.left.Main();} public static void main(String[] args){System.out.println(23);} }`,
	}
	t.Run("left", func(t *testing.T) { runCampaignCompilerProjectOracle(t, files, "entries.left.Main", "17\n") })
	t.Run("right", func(t *testing.T) { runCampaignCompilerProjectOracle(t, files, "entries.right.Main", "23\n") })
}
