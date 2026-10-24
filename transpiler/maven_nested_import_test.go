package transpiler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// A Java import may name a member type: p.Outer.Inner belongs to package p,
// not a package named p.Outer. The resolved source owner must drive both Go
// selectors and import paths, including before package-cycle collapsing.
func TestMavenNestedTypeImportJVMParity(t *testing.T) {
	for _, test := range []struct {
		name, pkg string
		cycle     bool
	}{
		{"same_package", "p", false}, {"cross_package", "q", false}, {"package_cycle", "q", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			suffix := `return "beta";`
			if test.cycle {
				suffix = `return q.Use.suffix();`
			}
			files := map[string]string{
				"pom.xml":                    `<project><groupId>example</groupId><artifactId>nested-import</artifactId><version>1</version></project>`,
				"src/main/java/p/Decoy.java": `package p;public final class Decoy {public static abstract class Inner {public abstract int unrelated();}}`,
				"src/main/java/p/Outer.java": fmt.Sprintf(`package p;
public final class Outer {
 private Outer(){}
 public interface Supplier {String get();}
 public static abstract class Inner {
  private final String key;
  protected Inner(String key){this.key=key;}
  public final String key(){return key;}
  public abstract int value();
 }
 public static String suffix(){%s}
}`, suffix),
				"src/main/java/" + test.pkg + "/Use.java": fmt.Sprintf(`package %s;
import p.Outer;import p.Outer.Inner;
public final class Use implements Outer.Supplier {
 private static final Inner ITEM=new Inner("alpha"){@Override public int value(){return 7;}};
 public static String suffix(){return "beta";}
 @Override public String get(){return ITEM.key()+":"+ITEM.value()+":"+Outer.suffix();}
 public static void main(String[] args){System.out.println(new Use().get());}
}`, test.pkg),
			}
			runNestedImportProjectOracle(t, files, test.pkg+".Use", test.cycle)
		})
	}
}

func runNestedImportProjectOracle(t *testing.T, files map[string]string, mainClass string, wantCycle bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	javac, err := campaignCompilerJavaTool("javac")
	if err != nil {
		t.Fatal(err)
	}
	java, err := campaignCompilerJavaTool("java")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	var sources []string
	for name, source := range files {
		path := filepath.Join(root, name)
		if err := writeProjectFile(path, []byte(source)); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(name, ".java") {
			sources = append(sources, path)
		}
	}
	sort.Strings(sources)
	classes := filepath.Join(root, "classes")
	if output, err := exec.CommandContext(ctx, javac, append([]string{"--release", "21", "-d", classes}, sources...)...).CombinedOutput(); err != nil {
		t.Fatalf("javac: %v\n%s", err, output)
	}
	want, err := exec.CommandContext(ctx, java, "-cp", classes, mainClass).CombinedOutput()
	if err != nil {
		t.Fatalf("JVM: %v\n%s", err, want)
	}
	if string(want) != "alpha:7:beta\n" {
		t.Fatalf("unexpected JVM oracle %q", want)
	}
	repo, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	generated := filepath.Join(root, "generated")
	compiler := exec.CommandContext(ctx, "go", "run", "./cmd/java2go", "-strict", "-maven", root, "-main-class", mainClass, "-runtime", repo, "-module", "nested.test/app", "-output", generated)
	compiler.Dir = repo
	if output, err := compiler.CombinedOutput(); err != nil {
		t.Fatalf("transpile: %v\n%s", err, output)
	}
	cycles, err := filepath.Glob(filepath.Join(generated, "java2go_scc_*"))
	if err != nil {
		t.Fatal(err)
	}
	if (len(cycles) > 0) != wantCycle {
		t.Fatalf("SCC output %v, want cycle=%v", cycles, wantCycle)
	}
	build := exec.CommandContext(ctx, "go", "build", "-mod=mod", "./...")
	build.Dir = generated
	build.Env = append(os.Environ(), "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("all generated packages: %v\n%s", err, output)
	}
	command := exec.CommandContext(ctx, "go", "run", "-mod=mod", "./cmd/app")
	command.Dir = generated
	command.Env = append(os.Environ(), "GOWORK=off")
	got, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("generated Go: %v\n%s", err, got)
	}
	if !bytes.Equal(want, got) {
		t.Fatalf("JVM %q != Go %q", want, got)
	}
}
