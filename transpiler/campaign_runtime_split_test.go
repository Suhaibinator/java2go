package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeSplitLimits(t *testing.T) {
	const source = `public class CampaignRuntimeSplitLimits {
    public static String run() {
        String[] negative = "a\tb\t".split("\t", -1);
        String[] bounded = "a:b:c:".split(":", 2);
        String[] zero = "a:b::".split(":", 0);
        String[] empty = "".split(",", 0);
        String[] leading = ":a:".split(":", -1);
        String[] zeroWidth = "abc".split("^", -1);
        return negative.length + ":" + negative[2] + ":" + bounded.length + ":" + bounded[1]
            + ":" + zero.length + ":" + empty.length + ":" + leading.length + ":" + leading[0]
            + ":" + zeroWidth.length + ":" + zeroWidth[0];
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeSplitLimits", source)
	t.Logf("JDK split oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestSplitOracle(t *testing.T) {
    if got := Run(); got != %q { t.Fatalf("JVM %%q != generated Go %%q", %q, got) }
}
`, want, want))
}
