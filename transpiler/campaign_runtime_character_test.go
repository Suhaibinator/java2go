package transpiler

import (
	"fmt"
	"testing"
	"unicode/utf16"
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
import (
    "slices"
    "testing"
    j "github.com/NickyBoy89/java2go/stdjava"
)
func TestDigitOracle(t *testing.T) {
    var got *j.JavaString = Run()
    if got == nil { t.Fatal("Run returned null") }
    units := got.UTF16Copy()
    if !slices.Equal(units, %#v) { t.Fatalf("JVM %%q != generated Go UTF16 %%#v", %q, units) }
}
`, utf16.Encode([]rune(want)), want))
}
