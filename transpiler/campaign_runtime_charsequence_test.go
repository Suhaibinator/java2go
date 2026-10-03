package transpiler

import (
	"fmt"
	"testing"
	"unicode/utf16"
)

func TestCampaignRuntimeCharSequence(t *testing.T) {
	const source = `public class CampaignRuntimeCharSequence {
    public static String run() {
        CharSequence text = "A😀Z";
        CharSequence builder = new StringBuilder("codec");
        return text.length() + ":" + (int) text.charAt(2)
            + ":" + text.equals("A😀Z")
            + ":" + builder.length() + ":" + builder.charAt(1)
            + ":" + "aBCd".regionMatches(true, 1, "bC", 0, 2)
            + ":" + "aBCd".regionMatches(1, "bc", 0, 2)
            + ":" + "😀Z".regionMatches(2, "Z", 0, 1)
            + ":" + "abc".regionMatches(0, "z", 0, -1)
            + ":" + "abc".regionMatches(-1, "z", 0, 1)
            + ":" + "𐐀".regionMatches(true, 0, "𐐨", 0, 2);
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeCharSequence", source)
	t.Logf("JDK CharSequence oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
    "slices"
    "testing"
    j "github.com/NickyBoy89/java2go/stdjava"
)
func TestCharSequenceOracle(t *testing.T) {
    var got *j.JavaString = Run()
    if got == nil { t.Fatal("Run returned null") }
    units := got.UTF16Copy()
    if !slices.Equal(units, %#v) { t.Fatalf("JVM %%q != generated Go UTF16 %%#v", %q, units) }
}
`, utf16.Encode([]rune(want)), want))
}
