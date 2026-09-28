package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeBitSet(t *testing.T) {
	const source = `import java.util.BitSet;
public class CampaignRuntimeBitSet {
    public static String run() {
        BitSet safe = new BitSet(256);
        safe.set('A'); safe.set('Z'); safe.set(130);
        BitSet copy = (BitSet) safe.clone();
        copy.set(2000);
        String negativeIndex = "missing";
        try { safe.get(-1); }
        catch (IndexOutOfBoundsException expected) { negativeIndex = "rejected"; }
        String negativeSize = "missing";
        try { new BitSet(-1); }
        catch (NegativeArraySizeException expected) { negativeSize = "rejected"; }
        return safe.get('A') + ":" + safe.get('B') + ":" + copy.get('Z')
            + ":" + copy.get(2000) + ":" + safe.get(2000) + ":" + (safe != copy)
            + ":" + safe.length() + ":" + safe.cardinality() + ":" + safe.size()
            + ":" + negativeIndex + ":" + negativeSize;
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeBitSet", source)
	t.Logf("JDK BitSet oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestBitSetOracle(t *testing.T) {
    if got := Run(); got != %q { t.Fatalf("JVM %%q != generated Go %%q", %q, got) }
}
`, want, want))
}
