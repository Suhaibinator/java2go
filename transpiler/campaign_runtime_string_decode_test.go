package transpiler

import (
	"fmt"
	"testing"
	"unicode/utf16"
)

func TestCampaignRuntimeStringMalformedDecoding(t *testing.T) {
	const source = `import java.nio.charset.*;
public class CampaignRuntimeStringMalformedDecoding {
 static String units(String text) {
  StringBuilder result = new StringBuilder();
  for (int i = 0; i < text.length(); i++) { result.append((int)text.charAt(i)).append(","); }
  return result.toString();
 }
 public static String run() throws Exception {
  byte[] truncated = new byte[]{-30,-126};
  byte[] surrogate = new byte[]{-19,-96,-128};
  return units(new String(truncated,StandardCharsets.UTF_8)) + ":"
   + units(new String(surrogate,StandardCharsets.UTF_8)) + ":"
   + units(new String(new byte[]{-30,-126,65},StandardCharsets.UTF_8)) + ":"
   + units(new String(new byte[]{-40,0,0,65},StandardCharsets.UTF_16BE)) + ":"
   + units(new String(new byte[]{0,-40,65,0},StandardCharsets.UTF_16LE)) + ":"
   + units(new String(new byte[]{-40,0,0},StandardCharsets.UTF_16BE)) + ":"
   + units(new String(new byte[]{-1,-2,65,0},StandardCharsets.UTF_16)) + ":"
   + units(new String(new byte[]{-2,-1,0,65},StandardCharsets.UTF_16BE)) + ":"
   + units(new String(truncated)) + ":" + units(new String(surrogate,"UTF-8"));
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeStringMalformedDecoding", source)
	t.Logf("JVM String decode oracle: %q", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import ("testing"; "slices")
func TestStringDecodeOracle(t *testing.T) {if got:=Run();got==nil||!slices.Equal(got.UTF16Copy(),%#v){t.Fatalf("JVM %%q != Go JavaString %%#v",%q,got)}}
`, utf16.Encode([]rune(want)), want))
}
