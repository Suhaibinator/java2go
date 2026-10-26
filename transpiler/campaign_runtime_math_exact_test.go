package transpiler

import (
	"fmt"
	"testing"
	"unicode/utf16"
)

func TestCampaignRuntimeMathAddExact(t *testing.T) {
	const source = `public class CampaignRuntimeMathExact {
    static String intCase(int left, int right) {
        try { return "ok=" + Math.addExact(left, right); }
        catch (ArithmeticException expected) { return "overflow"; }
    }
    static String longCase(long left, long right) {
        try { return "ok=" + Math.addExact(left, right); }
        catch (ArithmeticException expected) { return "overflow"; }
    }
    public static String run() {
        return intCase(Integer.MAX_VALUE, 0) + ":" + intCase(Integer.MAX_VALUE, 1)
            + ":" + intCase(Integer.MIN_VALUE, -1) + ":" + intCase(Integer.MIN_VALUE, Integer.MAX_VALUE)
            + ":" + longCase(Long.MAX_VALUE, 1L) + ":" + longCase(Long.MIN_VALUE, -1L)
            + ":" + longCase(Long.MIN_VALUE, Long.MAX_VALUE)
            + ":" + Math.addExact(Integer.MAX_VALUE, 1L);
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeMathExact", source)
	t.Logf("JDK addExact oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
    "slices"
    "testing"
)
func TestMathExactOracle(t *testing.T) {
    got := Run()
    if got == nil {
        t.Fatal("Run() returned null")
    }
    if units := got.UTF16Copy(); !slices.Equal(units, %#v) {
        t.Fatalf("JVM %%q != generated Go UTF16 %%v", %q, units)
    }
}
`, utf16.Encode([]rune(want)), want))
}
