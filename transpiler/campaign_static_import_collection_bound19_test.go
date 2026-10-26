package transpiler

import "testing"

// Applicability must infer E and validate its declared bound before selecting
// the generic Collection overload instead of the always-applicable Object one.
func TestCampaignStaticImportCollectionBoundedOverloadJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/library/Selectors.java": `package library;
import java.util.Collection;
public class Selectors {
 public static <E extends Number> int select(Collection<E> values) { return 1; }
 public static int select(Object value) { return 2; }
}`,
		"src/main/java/probe/Main.java": `package probe;
import static library.Selectors.select;
import java.util.ArrayList;
import java.util.List;
public class Main {
 public static void main(String[] args) {
  List<String> text = new ArrayList<String>(); text.add("text");
  System.out.println(select(text));
 }
}`,
	}, "probe.Main", "2\n")
}
