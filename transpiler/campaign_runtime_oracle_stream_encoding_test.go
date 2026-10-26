package transpiler

import "testing"

// The campaign compares raw output bytes. Its JVM oracle must preserve UTF-8
// output even when the test runner has an ASCII process locale.
func TestCampaignRuntimeJavaOracleUTF8Streams(t *testing.T) {
	const want = "éΩ😀"
	t.Run("stdout", func(t *testing.T) {
		const source = `public class CampaignUTF8Stdout {
    public static String run() { return "\u00e9\u03a9\ud83d\ude00"; }
}`
		if got := campaignRuntimeJavaOracle(t, "CampaignUTF8Stdout", source); got != want {
			t.Fatalf("JDK stdout bytes: got %q, want %q", got, want)
		}
	})
	t.Run("stderr", func(t *testing.T) {
		const source = `public class CampaignUTF8Stderr {
    public static String run() {
        System.err.print("\u00e9\u03a9\ud83d\ude00");
        return "";
    }
}`
		if got := campaignRuntimeJavaOracle(t, "CampaignUTF8Stderr", source); got != want {
			t.Fatalf("JDK stderr bytes: got %q, want %q", got, want)
		}
	})
}
