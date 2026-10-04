package transpiler

import (
	"os"
	"testing"
)

// Oracle fixtures must be captured from the frozen Java sources with JDK21
// before this test is run. Missing fixtures fail; no expected output is guessed.
func collectionBoundReview19Oracle(t *testing.T, name string) string {
	t.Helper()
	output, err := os.ReadFile("testdata/" + name + ".txt")
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}

func TestCampaignCollectionBoundReview19WildcardCapturePositiveJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/library/Selectors.java": `package library;
import java.util.Collection;
public class Selectors {
 public static <E extends Number> int select(Collection<E> values) { return 1; }
 public static int select(Object value) { return 2; }
 public static <E extends Number> int pair(Collection<E> first, Collection<E> second) { return 1; }
 public static int pair(Object first, Object second) { return 2; }
}
`,
		"src/main/java/probe/Main.java": `package probe;
import static library.Selectors.select;
import java.util.ArrayList;
import java.util.List;
public class Main {
 public static void main(String[] args) {
  List<? extends Integer> values = new ArrayList<Integer>();
  System.out.println(select(values));
 }
}
`,
	}, "probe.Main", collectionBoundReview19Oracle(t, "collection_bound_capture19_jdk21"))
}

func TestCampaignCollectionBoundReview19RepeatedInvariantJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": campaignStaticImportPOM26,
		"src/main/java/library/Selectors.java": `package library;
import java.util.Collection;
public class Selectors {
 public static <E extends Number> int select(Collection<E> values) { return 1; }
 public static int select(Object value) { return 2; }
 public static <E extends Number> int pair(Collection<E> first, Collection<E> second) { return 1; }
 public static int pair(Object first, Object second) { return 2; }
}
`,
		"src/main/java/probe/Main.java": `package probe;
import static library.Selectors.pair;
import java.util.ArrayList;
import java.util.List;
public class Main {
 public static void main(String[] args) {
  List<Integer> first = new ArrayList<Integer>();
  List<Double> different = new ArrayList<Double>();
  List<Integer> same = new ArrayList<Integer>();
  System.out.println(pair(first, different) + ":" + pair(first, same));
 }
}
`,
	}, "probe.Main", collectionBoundReview19Oracle(t, "collection_bound_repeated19_jdk21"))
}
