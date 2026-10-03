package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignModifiedVarargsJVMParity(t *testing.T) {
	const source = `public class ModifiedVarargs {
  private int size;
  public ModifiedVarargs(final int... values) { size = values.length; }
  public static int count(@Deprecated final String... values) { return values.length; }
  public static String run() { return "" + (new ModifiedVarargs(1,2,3).size * 100 + count("a", "b")); }
 }`
	want := campaignRuntimeJavaOracle(t, "ModifiedVarargs", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
    "slices"
    "testing"
    "unicode/utf16"
)
func TestModifiedVarargs(t *testing.T) {
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
