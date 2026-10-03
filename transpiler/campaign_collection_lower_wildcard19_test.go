package transpiler

import (
	"os"
	"testing"
)

func TestCampaignCollectionLowerWildcardInference19JVM(t *testing.T) {
	oracle, err := os.ReadFile("testdata/collection_lower_wildcard19_jdk21.txt")
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/library/Selectors.java": `package library;
import java.util.Collection;
public class Selectors {
 public static int trace;
 public static <T> int select(Collection<? super T> values) { trace = trace * 10 + 1; return 1; }
 public static int select(Object value) { trace = trace * 10 + 2; return 2; }
}
`,
		"src/main/java/probe/Main.java": `package probe;
import static library.Selectors.select;
import java.util.ArrayList;
import java.util.List;
import library.Selectors;
public class Main {
 public static void main(String[] args) {
  List<String> strings = new ArrayList<String>(); strings.add("retained");
  List<Object> objects = new ArrayList<Object>(); objects.add(strings);
  int first = select(strings);
  int second = select(objects);
  System.out.println(first + ":" + second + ":" + Selectors.trace + ":" + strings.get(0) + ":" + (objects.get(0) == strings));
 }
}
`,
	}, "probe.Main", string(oracle))
}
