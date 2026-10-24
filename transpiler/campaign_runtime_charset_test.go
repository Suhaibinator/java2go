package transpiler

import (
	"fmt"
	"testing"
	"unicode/utf16"
)

func TestCampaignRuntimeCharsets(t *testing.T) {
	const source = `import java.nio.charset.Charset;
import java.nio.charset.StandardCharsets;
public class CampaignRuntimeCharsets {
    public static String run() {
        String value = "café😀";
        byte[] encoded = value.getBytes(StandardCharsets.UTF_8);
        String decoded = new String(encoded, Charset.forName("utf8"));
        char[] chars = value.toCharArray();
        String copied = new String(chars);
        return StandardCharsets.US_ASCII.name() + ":" + StandardCharsets.ISO_8859_1.name()
            + ":" + StandardCharsets.UTF_8.name() + ":" + StandardCharsets.UTF_16BE.name()
            + ":" + StandardCharsets.UTF_16LE.name() + ":" + StandardCharsets.UTF_16.name()
            + ":" + encoded.length + ":" + decoded + ":" + chars.length + ":" + copied
            + ":" + new String(value.getBytes(StandardCharsets.ISO_8859_1), StandardCharsets.ISO_8859_1)
            + ":" + new String(value.getBytes(StandardCharsets.UTF_16), StandardCharsets.UTF_16);
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeCharsets", source)
	t.Logf("JDK charset oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import ("testing"; "slices")
func TestCharsetsOracle(t *testing.T) {
    if got := Run(); got == nil || !slices.Equal(got.UTF16Copy(), %#v) { t.Fatalf("JVM %%q != generated JavaString %%#v", %q, got) }
}
`, utf16.Encode([]rune(want)), want))
}

func TestCampaignRuntimeCharsetNames(t *testing.T) {
	const source = `public class CampaignRuntimeCharsetNames {
    public static String run() throws Exception {
        byte[] data = "café".getBytes("utf8");
        String encodedError = "missing";
        String decodedError = "missing";
        try { "text".getBytes("not-a-charset"); }
        catch (java.io.UnsupportedEncodingException expected) { encodedError = "unsupported"; }
        try { new String(data, "not-a-charset"); }
        catch (java.io.UnsupportedEncodingException expected) { decodedError = "unsupported"; }
        return new String(data, "UTF-8") + ":" + encodedError + ":" + decodedError;
    }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeCharsetNames", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import ("testing"; "slices")
func TestCharsetNamesOracle(t *testing.T) {
    if got := Run(); got == nil || !slices.Equal(got.UTF16Copy(), %#v) { t.Fatalf("JVM %%q != generated JavaString %%#v", %q, got) }
}
`, utf16.Encode([]rune(want)), want))
}
