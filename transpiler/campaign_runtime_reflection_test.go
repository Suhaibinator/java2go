package transpiler

import (
	"fmt"
	"testing"
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
import "testing"
func TestClassNameOracle(t *testing.T) {
    if got := Run(); got != %q { t.Fatalf("JVM %%q != generated Go %%q", %q, got) }
}
`, want, want))
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
import "testing"
func TestBuilderChainOracle(t *testing.T) {
    if got := Run(); got != %q { t.Fatalf("JVM %%q != generated Go %%q", %q, got) }
}
`, want, want))
}
