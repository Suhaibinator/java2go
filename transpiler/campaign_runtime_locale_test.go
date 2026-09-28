package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeLocaleCase(t *testing.T) {
	const source = `import java.util.Locale;
public class CampaignRuntimeLocaleCase {
    public static String run() {
        Locale turkish = Locale.forLanguageTag("tr");
        return "straße ﬃ".toUpperCase(Locale.ROOT)
            + ":" + "ΟΣ".toLowerCase(Locale.ROOT)
            + ":" + "iIıİ".toUpperCase(turkish)
            + ":" + "iIıİ".toLowerCase(turkish)
            + ":" + "İ".toLowerCase(Locale.ROOT);
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeLocaleCase", source)
	t.Logf("JDK locale oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestLocaleOracle(t *testing.T) {
    if got := Run(); got != %q { t.Fatalf("JVM %%q != generated Go %%q", %q, got) }
}
`, want, want))
}
