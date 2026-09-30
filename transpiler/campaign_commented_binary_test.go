package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignCommentedBinaryJVMParity(t *testing.T) {
	const source = `public class CampaignCommentedBinary{
 static int calls;
 static boolean touch(){calls++;return true;}
 public static String run(){boolean a=true;boolean b=false;boolean first=(a // operand comment
 && b);boolean second=(b && /* before right */touch());long shifted=(-8L /* shift */ >>> /* distance */ 2);var text="sum=" /* concatenation */ + (1 /* add */ + 2);return first+":"+second+":"+calls+":"+shifted+":"+text;}
}`
	want := campaignRuntimeJavaOracle(t, "CampaignCommentedBinary", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import (
 "slices"
 "testing"
 "unicode/utf16"
 j "github.com/NickyBoy89/java2go/stdjava"
)
func TestCommentedBinary(t *testing.T) {
 var got *j.JavaString = Run()
 if got == nil { t.Fatal("Run returned null; JVM returned a String value") }
 want := utf16.Encode([]rune(%q))
 if units := got.UTF16Copy(); !slices.Equal(units, want) {
  t.Fatalf("JVM UTF16 %%x != Go UTF16 %%x", want, units)
 }
}`, want))
}
