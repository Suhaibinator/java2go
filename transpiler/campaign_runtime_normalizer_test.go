package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeNormalizer(t *testing.T) {
	const source = `import java.text.Normalizer;
public class CampaignRuntimeNormalizer {
    public static String run() {
        String decomposed = "e\u0301";
        String composed = Normalizer.normalize(decomposed, Normalizer.Form.NFC);
        String compatibility = Normalizer.normalize("Ａﬃ①", Normalizer.Form.NFKC);
        String canonical = Normalizer.normalize(composed, Normalizer.Form.NFD);
        String fromBuilder = Normalizer.normalize(new StringBuilder(decomposed), Normalizer.Form.NFC);
        return composed + ":" + composed.length() + ":" + compatibility + ":" + canonical.length()
            + ":" + Normalizer.isNormalized(decomposed, Normalizer.Form.NFC)
            + ":" + Normalizer.isNormalized(canonical, Normalizer.Form.NFD)
            + ":" + Normalizer.normalize("①", Normalizer.Form.NFKD) + ":" + fromBuilder;
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeNormalizer", source)
	t.Logf("JDK normalization oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestNormalizerOracle(t *testing.T) {
    if got := Run(); got != %q { t.Fatalf("JVM %%q != generated Go %%q", %q, got) }
}
`, want, want))
}
