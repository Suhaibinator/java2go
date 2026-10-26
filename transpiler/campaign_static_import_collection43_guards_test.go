package transpiler

import "testing"

func TestCampaignStaticImportCollection43InvariantGuardJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                              campaignStaticImportPOM26,
		"src/main/java/library/Selection.java": `package library;public class Selection {public static int select(java.util.Collection<Object> values){return 1;}public static int select(Object value){return 2;}}`,
		"src/main/java/probe/Main.java":        `package probe;import static library.Selection.select;import java.util.ArrayList;import java.util.List;public class Main {public static void main(String[] args){List<String> values=new ArrayList<>();values.add("x");System.out.println(select(values));}}`,
	}, "probe.Main", "2\n")
}
func TestCampaignStaticImportCollection43SourceShadowGuardJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                               campaignStaticImportPOM26,
		"src/main/java/foreign/Collection.java": `package foreign;public class Collection<E>{}`,
		"src/main/java/library/Selection.java":  `package library;public class Selection {public static <E> int select(foreign.Collection<E> values){return 1;}public static int select(Object value){return 2;}}`,
		"src/main/java/probe/Main.java":         `package probe;import static library.Selection.select;import java.util.ArrayList;import java.util.List;public class Main {public static void main(String[] args){List<String> values=new ArrayList<>();values.add("x");System.out.println(select(values));}}`,
	}, "probe.Main", "2\n")
}
