package transpiler

import (
	"fmt"
	"testing"
	"unicode/utf16"
)

func TestCampaignRuntimeClassSimpleName(t *testing.T) {
	const source = `public class CampaignRuntimeClassName {
    static class Nested {}
    public static String run() {
        Exception wrapped = new Exception("wrapped", new IllegalArgumentException("cause"));
        Object array = new int[2][3];
        return wrapped.getCause().getClass().getSimpleName() + ":" + Nested.class.getSimpleName()
            + ":" + array.getClass().getSimpleName() + ":" + String.class.getSimpleName();
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeClassName", source)
	t.Logf("JDK class-name oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import ("testing"; "slices")
func TestClassNameOracle(t *testing.T) {
    if got := Run(); got == nil || !slices.Equal(got.UTF16Copy(), %#v) { t.Fatalf("JVM %%q != generated JavaString %%#v", %q, got) }
}
`, utf16.Encode([]rune(want)), want))
}

func TestCampaignRuntimeStringBuilderChain(t *testing.T) {
	const source = `public class CampaignRuntimeBuilderChain {
    public static String run() {
        return new StringBuilder().append("sum").append('=').append(42).append(':').append(true).toString();
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeBuilderChain", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import ("testing"; "slices")
func TestBuilderChainOracle(t *testing.T) {
    if got := Run(); got == nil || !slices.Equal(got.UTF16Copy(), %#v) { t.Fatalf("JVM %%q != generated JavaString %%#v", %q, got) }
}
`, utf16.Encode([]rune(want)), want))
}
