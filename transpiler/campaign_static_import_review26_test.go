package transpiler

import "testing"

func TestCampaignStaticImportReview26IntrinsicApplicabilityJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                        campaignStaticImportPOM26,
		"src/main/java/library/Ops.java": `package library;public class Ops {public static int nanoTime(int value){return value+1;}}`,
		"src/main/java/probe/Main.java": `package probe;import static java.lang.System.nanoTime;import static library.Ops.nanoTime;
public class Main {public static void main(String[] args){long start=nanoTime();int value=nanoTime(7);long end=nanoTime();System.out.println(value+":"+(end-start>=0L));}}`,
	}, "probe.Main", "8:true\n")
}

func TestCampaignStaticImportReview26HiddenStaticJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                          campaignStaticImportPOM26,
		"src/main/java/library/Base.java":  `package library;public class Base {public static int value(){return 1;}}`,
		"src/main/java/library/Child.java": `package library;public class Child extends Base {public static int value(){return 2;}}`,
		"src/main/java/probe/Main.java": `package probe;import static library.Child.value;
public class Main {public static void main(String[] args){System.out.println(value());}}`,
	}, "probe.Main", "2\n")
}

func TestCampaignStaticImportReview26PrivateAncestorJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                         campaignStaticImportPOM26,
		"src/main/java/library/Base.java": `package library;public class Base {private int value(){return -1;}}`,
		"src/main/java/library/Ops.java":  `package library;public class Ops {public static int value(){return 3;}}`,
		"src/main/java/probe/Main.java": `package probe;import static library.Ops.value;
public class Main extends library.Base {public static void main(String[] args){System.out.println(value());}}`,
	}, "probe.Main", "3\n")
}
