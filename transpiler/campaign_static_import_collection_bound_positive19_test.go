package transpiler

import "testing"

func TestCampaignStaticImportCollectionBoundPositiveAndOwnerJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/library/Selectors.java": `package library;
import java.util.Collection;
public class Selectors {
 public static <E extends Number> int select(Collection<E> values) { return 1; }
 public static int select(Object value) { return 2; }
}`,
		"src/main/java/probe/Number.java": `package probe; public class Number {}`,
		"src/main/java/probe/Main.java": `package probe;
import static library.Selectors.select;
import java.util.ArrayList;
import java.util.List;
public class Main {
 static <Number> int fromShadowedCaller(Number ignored) {
  List<Integer> values = new ArrayList<Integer>(); values.add(7);
  return select(values);
 }
 public static void main(String[] args) {
  List<Integer> values = new ArrayList<Integer>(); values.add(8);
  System.out.println(select(values) + ":" + fromShadowedCaller("caller"));
 }
}`,
	}, "probe.Main", "1:1\n")
}
