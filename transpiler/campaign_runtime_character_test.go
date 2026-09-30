package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeCharacterDigit(t *testing.T) {
	const source = `public class CampaignRuntimeCharacterDigit {
    public static String run() {
        return Character.digit('f', 16) + ":" + Character.digit('Ｆ', 16)
            + ":" + Character.digit('９', 10) + ":" + Character.digit('٣', 10)
            + ":" + Character.digit('७', 10) + ":" + Character.digit(0x1D7DD, 10)
            + ":" + Character.digit('²', 10) + ":" + Character.digit('z', 35)
            + ":" + Character.digit('z', 36) + ":" + Character.digit('1', 1)
            + ":" + Character.digit('1', 37)
            + ":" + (int) Character.forDigit(15, 16) + ":" + (int) Character.forDigit(35, 36)
            + ":" + (int) Character.forDigit(16, 16) + ":" + (int) Character.forDigit(-1, 16)
            + ":" + (int) Character.forDigit(1, 37);
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeCharacterDigit", source)
	t.Logf("JDK digit oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestDigitOracle(t *testing.T) {
    if got := Run(); got != %q { t.Fatalf("JVM %%q != generated Go %%q", %q, got) }
}
`, want, want))
}
