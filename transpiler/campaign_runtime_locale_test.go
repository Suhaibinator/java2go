package transpiler

import (
	"fmt"
	"testing"
	"unicode/utf16"
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
import (
    "slices"
    "testing"
    j "github.com/NickyBoy89/java2go/stdjava"
)
func TestLocaleOracle(t *testing.T) {
    var got *j.JavaString = Run()
    if got == nil { t.Fatal("Run returned null") }
    units := got.UTF16Copy()
    if !slices.Equal(units, %#v) { t.Fatalf("JVM %%q != generated Go UTF16 %%#v", %q, units) }
}
`, utf16.Encode([]rune(want)), want))
}
