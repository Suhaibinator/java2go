package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignForMultipleDeclaratorsJVMParity(t *testing.T) {
	const source = `public class CampaignForInitializers {
 private static int trace;
 private static int next(int value) { trace = trace * 10 + value; return value; }
 public static String run() {
  trace = 0;
  int total = 0;
  for (int i = next(1), j = i + next(2), k = next(j); i < 4; i++) {
   total += j + k;
  }
  int j = 99;
  for (int i = next(4), unused = next(5); i < 0; i++) { total += i; }
  outer: for (int i = 0, offset = 10; i < 4; i++) {
   if (i == 1) { continue outer; }
   total += i + offset;
   if (i == 3) { break outer; }
  }
  for (long i = 2147483647L, end = i + 2L; i < end; i++) { total += (int) (i - 2147483647L); }
  return total + ":" + trace + ":" + j;
 }
}`
	want := campaignRuntimeJavaOracle(t, "CampaignForInitializers", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
 j "github.com/NickyBoy89/java2go/stdjava"
)
func TestForInitializers(t *testing.T) {
 var got *j.JavaString = Run()
 if got == nil { t.Fatal("Run returned null; JVM returned a String value") }
 want := utf16.Encode([]rune(%q))
 if units := got.UTF16Copy(); !slices.Equal(units, want) {
  t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x", want, units)
 }
}`, want))
}
