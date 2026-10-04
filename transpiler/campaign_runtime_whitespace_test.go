package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignRuntimeJavaWhitespace(t *testing.T) {
	const source = `public class CampaignWhitespace {
 public static String run() {
  int[] points = {0, 1, 8, 9, 10, 11, 12, 13, 14, 27, 28, 29, 30, 31, 32,
    133, 160, 5760, 8192, 8193, 8194, 8195, 8196, 8197, 8198, 8199, 8200, 8201,
    8202, 8203, 8232, 8233, 8239, 8287, 12288, 65279};
  StringBuilder result = new StringBuilder();
  result.append("empty:").append("".isBlank()).append(":").append("".trim().length()).append(":").append("".strip().length()).append(";");
  for (int point : points) {
   String unit = new String(new char[] {(char)point});
   String around = unit + "x" + unit;
   result.append(point).append(":").append(around.trim().length()).append(":")
      .append(around.strip().length()).append(":").append(unit.isBlank()).append(":")
      .append(Character.isWhitespace(point)).append(":").append((unit + unit).strip().length()).append(";");
  }
  result.append("all:");
  for (int point = 0; point <= 0x10ffff; point++) {
   if (Character.isWhitespace(point)) { result.append(point).append(","); }
  }
  return result.toString();
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignWhitespace", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
    "slices"
    "testing"
    "unicode/utf16"
)
func TestWhitespace(t *testing.T) {
    got := Run()
    if got == nil {
        t.Fatal("Run() returned null")
    }
    const want = %q
    if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune(want))) {
        t.Fatalf("JVM %%q != Go UTF16 %%v", want, units)
    }
}`, want))
}
