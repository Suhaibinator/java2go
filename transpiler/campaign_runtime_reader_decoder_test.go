package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeReaderMalformedDecoding(t *testing.T) {
	const source = `import java.io.*;
import java.nio.charset.*;
public class CampaignRuntimeReaderMalformedDecoding {
 static String decode(byte[] bytes, Charset charset) throws Exception {
  BufferedReader reader = new BufferedReader(new InputStreamReader(new ByteArrayInputStream(bytes), charset));
  String text = reader.readLine();
  StringBuilder result = new StringBuilder();
  for (int i = 0; i < text.length(); i++) { result.append((int)text.charAt(i)).append(","); }
  reader.close();
  return result.toString();
 }
 public static String run() throws Exception {
  return decode(new byte[]{-19,-96,-128},StandardCharsets.UTF_8) + ":"
   + decode(new byte[]{-30,-126,65},StandardCharsets.UTF_8) + ":"
   + decode(new byte[]{-30,-126},StandardCharsets.UTF_8) + ":"
   + decode(new byte[]{-40,0,0,65},StandardCharsets.UTF_16BE) + ":"
   + decode(new byte[]{0,-40,65,0},StandardCharsets.UTF_16LE) + ":"
   + decode(new byte[]{-1,-2,65,0},StandardCharsets.UTF_16);
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignRuntimeReaderMalformedDecoding", source)
	t.Logf("JVM decoder oracle: %q", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestDecoderOracle(t *testing.T) {if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}
`, want, want))
}
