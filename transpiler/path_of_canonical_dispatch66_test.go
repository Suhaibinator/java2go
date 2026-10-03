package transpiler

import "testing"

// Keep Java Path.of/Paths.get declaration identity, canonical String overloads,
// evaluation order and null behavior tied to a fresh JVM before strict race Go.
func TestCampaignPathOfCanonicalDispatchJVM66(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project xmlns="http://maven.apache.org/POM/4.0.0"><modelVersion>4.0.0</modelVersion><groupId>campaign</groupId><artifactId>path-dispatch</artifactId><version>1</version><properties><maven.compiler.release>21</maven.compiler.release><project.build.sourceEncoding>UTF-8</project.build.sourceEncoding></properties></project>
`,
		"src/main/java/app/Main.java": `package app;
import java.nio.file.Path;
import java.nio.file.Paths;
public final class Main {
  static int calls;
  static String next(String value) { calls++; return value; }
  public static void main(String[] args) {
    String first = args.length == 0 ? "alpha" : args[0];
    System.out.println(Path.of(first, "beta").toString());
    System.out.println(java.nio.file.Path.of("gamma").toString());
    System.out.println(Paths.get("delta").toString());
    String[] more = new String[]{"two", "three"};
    System.out.println(Path.of("one", more).toString());
    System.out.println(Path.of(next("a"), next("b")).toString() + ":" + calls);
    try { Path.of(next(null), next("late")); }
    catch (NullPointerException expected) { System.out.println("null:" + calls); }
    try { Path.of("base", (String[])null); }
    catch (NullPointerException expected) { System.out.println("array-null"); }
    System.out.println(p.Path.of("x", "y") + ":" + p.Paths.get("z"));
  }
}
`,
		"src/main/java/p/Path.java": `package p;
public final class Path {
  public static String of(String first, String more) { return "source"; }
}
`,
		"src/main/java/p/Paths.java": `package p;
public final class Paths {
  public static String get(String first) { return "sourcePaths"; }
}
`,
	}
	runCampaignCompilerStrictProjectOracle(t, files, "app.Main", "alpha/beta\ngamma\ndelta\none/two/three\na/b:2\nnull:4\narray-null\nsource:sourcePaths\n")
}
