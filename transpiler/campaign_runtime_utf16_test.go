package transpiler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// This oracle uses supplementary characters as present in real JSON records and
// UTF-8 codec inputs. String observations must use Java's UTF-16 code units.
func TestCampaignRuntimeUTF16Observations(t *testing.T) {
	const source = `public class CampaignRuntimeUTF16 {
        public static String run() {
            String text = "A😀Z😀";
            return text.length() + ":" + (int) text.charAt(1) + ":" + (int) text.charAt(2)
                + ":" + text.indexOf("Z") + ":" + text.lastIndexOf("😀")
                + ":" + text.chars().sum() + ":" + "😀".compareTo("\uE000")
                + ":" + "a".compareTo("z") + ":" + "ab".compareTo("abcdef");
        }
    }`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeUTF16", source)
	t.Logf("JDK UTF-16 oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestUTF16Oracle(t *testing.T) {
    if got := Run(); got != %q { t.Fatalf("JVM %%q != generated Go %%q", %q, got) }
}
`, want, want))
}

func TestCampaignRuntimeUTF16NaturalOrdering(t *testing.T) {
	const source = `import java.util.ArrayList;
import java.util.List;
import java.util.Arrays;
import java.util.Collections;
import java.util.Comparator;
public class CampaignRuntimeUTF16Ordering {
    public static String run() {
        List<String> values = new ArrayList<String>();
        values.add("\uE000");
        values.add("😀");
        String streamFirst = values.stream().sorted().findFirst().get();
        Collections.sort(values);
        String[] array = {"\uE000", "😀"};
        Arrays.sort(array);
        Comparator<String> natural = Comparator.naturalOrder();
        return values.get(0) + ":" + streamFirst + ":" + array[0]
            + ":" + natural.compare("😀", "\uE000")
            + ":" + natural.compare("a", "z");
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeUTF16Ordering", source)
	t.Logf("JDK natural-order oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestUTF16OrderingOracle(t *testing.T) {
    if got := Run(); got != %q { t.Fatalf("JVM %%q != generated Go %%q", %q, got) }
}
`, want, want))
}

func TestCampaignRuntimeOraclePreservesWhitespace(t *testing.T) {
	const source = `public class CampaignWhitespaceOracle {
  public static String run() { return " \tvalue\n "; }
 }`
	if got := campaignRuntimeJavaOracle(t, "CampaignWhitespaceOracle", source); got != " \tvalue\n " {
		t.Fatalf("oracle changed observable whitespace: %q", got)
	}
}

func campaignRuntimeJavaOracle(t *testing.T, name, source string) string {
	t.Helper()
	home := os.Getenv("JAVA_HOME")
	javac, java := "javac", "java"
	if home != "" {
		javac, java = filepath.Join(home, "bin", "javac"), filepath.Join(home, "bin", "java")
	}
	for _, tool := range []*string{&javac, &java} {
		resolved, err := exec.LookPath(*tool)
		if err != nil {
			t.Fatalf("campaign requires JDK 21: cannot find %s (set JAVA_HOME): %v", *tool, err)
		}
		*tool = resolved
	}
	dir := t.TempDir()
	driver := `class CampaignRuntimeOracle { public static void main(String[] args) throws Exception { System.out.print(` + name + `.run()); } }`
	path := filepath.Join(dir, name+".java")
	if err := os.WriteFile(path, []byte(source+"\n"+driver), 0600); err != nil {
		t.Fatal(err)
	}
	compileContext, cancelCompile := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelCompile()
	if out, err := exec.CommandContext(compileContext, javac, "--release", "21", "-encoding", "UTF-8", "-d", dir, path).CombinedOutput(); err != nil {
		t.Fatalf("JDK 21 oracle compilation: %v\n%s", err, out)
	}
	runContext, cancelRun := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelRun()
	out, err := exec.CommandContext(runContext, java, "-cp", dir, "CampaignRuntimeOracle").CombinedOutput()
	if err != nil {
		t.Fatalf("JDK oracle execution: %v\n%s", err, out)
	}
	return string(out)
}
