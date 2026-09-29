package transpiler

import "testing"

func TestCampaignStaticImportHiddenType26CanonicalJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                          campaignStaticImportPOM26,
		"src/main/java/library/Base.java":  `package library;public class Base {public static int value(java.lang.String text){return 1;}}`,
		"src/main/java/library/Child.java": `package library;public class Child extends Base {public static int value(String text){return 2;}}`,
		"src/main/java/probe/Main.java": `package probe;import static library.Child.value;
public class Main {public static void main(String[] args){System.out.println(value("x"));}}`,
	}, "probe.Main", "2\n")
}

func TestCampaignStaticImportHiddenType26SourceShadowJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                          campaignStaticImportPOM26,
		"src/main/java/shadow/String.java": `package shadow;public class String {}`,
		"src/main/java/library/Base.java":  `package library;public class Base {public static int value(java.lang.String text){return 1;}}`,
		"src/main/java/library/Child.java": `package library;import shadow.String;public class Child extends Base {public static int value(String text){return 2;}}`,
		"src/main/java/probe/Main.java": `package probe;import static library.Child.value;
public class Main {public static void main(String[] args){System.out.println(value("x")+":"+value(new shadow.String()));}}`,
	}, "probe.Main", "1:2\n")
}
