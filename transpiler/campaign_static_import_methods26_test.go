package transpiler

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

const campaignStaticImportPOM26 = `<project><modelVersion>4.0.0</modelVersion><groupId>staticimports</groupId><artifactId>probe</artifactId><version>1</version></project>`

func TestCampaignStaticImport26CanonicalWildcardJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;import static java.lang.System.*;
class System {static long nanoTime(){return -17L;}}
public class Main {
 static String kind(long value){return "long";}static String kind(double value){return "double";}
 public static void main(String[] args){long first=nanoTime();long last=nanoTime();java.lang.System.out.println((last-first>=0L)+":"+kind(nanoTime())+":"+System.nanoTime());}
}`,
	}, "probe.Main", "true:long:-17\n")
}

func TestCampaignStaticImport26SourceOverloadsJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/library/Ops.java": `package library;
public class Ops {
 public static String select(int value){return "int:"+value;}
 public static String select(long value){return "long:"+value;}
 public static int sum(int... values){int result=0;for(int value:values)result+=value;return result;}
}`,
		"src/main/java/probe/Main.java": `package probe;import static library.Ops.select;import static library.Ops.select;import static library.Ops.*;
public class Main {
 static String trace="";static int mark(int value){trace+=value;return value;}
 public static void main(String[] args){String one=select(mark(1));String two=select((long)mark(2));int total=sum(mark(3),mark(4),mark(5));System.out.println(trace+":"+one+":"+two+":"+total);}
}`,
	}, "probe.Main", "12345:int:1:long:2:12\n")
}

func TestCampaignStaticImport26LexicalAndImportPrecedenceJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                          campaignStaticImportPOM26,
		"src/main/java/library/One.java":   `package library;public class One {public static int value(){return 1;}public static int own(){return -1;}public static int inherited(){return -2;}public static int enclosing(){return -3;}}`,
		"src/main/java/library/Other.java": `package library;public class Other {public static int value(){return 99;}}`,
		"src/main/java/probe/Main.java": `package probe;import static library.One.*;import static library.One.value;import static library.Other.*;
class Base {static int inherited(){return 7;}}
public class Main extends Base {
 static int own(){return 6;}static int enclosing(){return 8;}
 static class Nested {static int call(){return enclosing();}}
 public static void main(String[] args){System.out.println(own()+":"+inherited()+":"+Nested.call()+":"+value());}
}`,
	}, "probe.Main", "6:7:8:1\n")
}

// Invalid import/lexical cases require rejection, not an execution oracle.
// javac establishes invalidity before strict translation is checked.
func TestCampaignStaticImport26RejectAmbiguity(t *testing.T) {
	for _, tc := range []struct{ name, imports, members, call string }{
		{"explicit", "import static library.One.value;import static library.Other.value;", "", "value()"},
		{"wildcard", "import static library.One.*;import static library.Other.*;", "", "value()"},
		{"lexical-name", "import static library.One.value;", "static int value(String text){return 3;}", "value()"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			javac, err := campaignCompilerJavaTool("javac")
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			files := map[string]string{
				"pom.xml":                          campaignStaticImportPOM26,
				"src/main/java/library/One.java":   `package library;public class One {public static int value(){return 1;}}`,
				"src/main/java/library/Other.java": `package library;public class Other {public static int value(){return 2;}}`,
				"src/main/java/probe/Main.java":    "package probe;" + tc.imports + "public class Main {" + tc.members + "public static void main(String[] args){System.out.println(" + tc.call + ");}}",
			}
			var sources []string
			for name, source := range files {
				path := filepath.Join(root, name)
				if err := writeProjectFile(path, []byte(source)); err != nil {
					t.Fatal(err)
				}
				if filepath.Ext(path) == ".java" {
					sources = append(sources, path)
				}
			}
			sort.Strings(sources)
			run := func(name string, args ...string) ([]byte, error) {
				ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, name, args...)
				cmd.Dir = repoRootDir(t)
				cmd.Env = append(os.Environ(), "GOWORK=off")
				cmd.WaitDelay = 5 * time.Second
				out, err := cmd.CombinedOutput()
				if ctx.Err() != nil {
					t.Fatalf("command timeout: %s: %v", name, ctx.Err())
				}
				return out, err
			}
			out, err := run(javac, append([]string{"--release", "21", "-encoding", "UTF-8", "-d", filepath.Join(root, "classes")}, sources...)...)
			if err == nil {
				t.Fatalf("invalidity prerequisite failed: JVM accepted source: %s", out)
			}
			t.Logf("JDK rejection: %s", out)
			compiler := filepath.Join(root, "java2go")
			if out, err = run("go", "build", "-o", compiler, "./cmd/java2go"); err != nil {
				t.Fatalf("compiler build: %v: %s", err, out)
			}
			out, err = run(compiler, "-strict", "-maven", root, "-main-class", "probe.Main", "-runtime", repoRootDir(t), "-output", filepath.Join(root, "generated"))
			if err == nil {
				t.Fatalf("strict compiler accepted JVM-rejected static method resolution: %s", out)
			}
			t.Logf("strict rejection: %s", out)
		})
	}
}

func TestCampaignStaticImport26CanonicalWildcardEffectsJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/probe/Main.java": `package probe;import static java.lang.System.*;
class System {static int calls;static long nanoTime(){calls++;return -17L;}}
public class Main {public static void main(String[] args){long first=nanoTime();long last=nanoTime();long source=System.nanoTime();java.lang.System.out.println((last-first>=0L)+":"+source+":"+System.calls);}}`,
	}, "probe.Main", "true:-17:1\n")
}
