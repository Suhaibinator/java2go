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
 import "testing"
 func TestModifiedVarargs(t *testing.T) { if got:=Run(); got != %q { t.Fatalf("got %%s",got) } }
 `, want))
}
