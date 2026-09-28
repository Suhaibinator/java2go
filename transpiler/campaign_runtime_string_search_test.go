package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeStringSearchOverloads(t *testing.T) {
	const source = `public class CampaignStringSearch {
 public static String run() {
  String text = "A😀|møøse|😀Z";
  int[] points = {'|', 0x1f600, 0xd83d, 0xde00, -1, 0x110000, 'x'};
  int[] starts = {-10, 0, 1, 2, 3, 9, 10, 11, 12, 100};
  StringBuilder result = new StringBuilder();
  for (int point : points) {
   result.append(text.indexOf(point)).append(":").append(text.lastIndexOf(point)).append(";");
   for (int start : starts) {
    result.append(text.indexOf(point,start)).append(":").append(text.lastIndexOf(point,start)).append(";");
   }
  }
  String[] words = {"", "😀", "|", "møøse", "Z", "missing"};
  for (String word : words) {
   for (int start : starts) {
    result.append(text.indexOf(word,start)).append(":").append(text.lastIndexOf(word,start)).append(";");
   }
  }
  Integer boxed = 0x1f600;
  Integer from = 2;
  Character separator = '|';
  result.append(text.indexOf(boxed,from)).append(":").append(text.lastIndexOf(separator));
  try { String missing=null; text.indexOf(missing); } catch (NullPointerException expected) { result.append(":null-string"); }
  try { Integer missing=null; text.indexOf(missing); } catch (NullPointerException expected) { result.append(":null-integer"); }
  return result.toString();
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignStringSearch", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import "testing"
func TestSearch(t *testing.T) {
 if got := Run(); got != %q { t.Fatalf("JVM %%q != Go %%q", %q, got) }
}
`, want, want))
}
