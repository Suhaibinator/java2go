package transpiler

import "testing"

func TestCampaignImportKind44StaticNestedExplicitJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                          campaignStaticImportPOM26,
		"src/main/java/library/Outer.java": `package library;public class Outer{public static class Nested{public int id;public Nested(int id){this.id=id;}}}`,
		"src/main/java/probe/Main.java":    `package probe;import static library.Outer.Nested;public class Main{public static void main(String[]args){Nested value=new Nested(7);System.out.print(value.id);}}`,
	}, "probe.Main", "7")
}
func TestCampaignImportKind44SingleTypePrecedenceJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                       campaignStaticImportPOM26,
		"src/main/java/probe/Math.java": `package probe;public class Math{public static int toIntExact(long value){return 17;}}`,
		"src/main/java/probe/Main.java": `package probe;import java.lang.Math;import static java.lang.Math.*;public class Main{public static void main(String[]args){System.out.print(Math.toIntExact(3)+":"+toIntExact(4));}}`,
	}, "probe.Main", "3:4")
}
func TestCampaignImportKind44StaticNestedWildcardJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                          campaignStaticImportPOM26,
		"src/main/java/library/Outer.java": `package library;public class Outer{public static class Nested{public int id;public Nested(int id){this.id=id;}}}`,
		"src/main/java/probe/Main.java":    `package probe;import static library.Outer.*;public class Main{public static void main(String[]args){Nested value=new Nested(9);System.out.print(value.id);}}`,
	}, "probe.Main", "9")
}
