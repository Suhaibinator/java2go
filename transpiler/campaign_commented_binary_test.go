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
import "testing"
func TestCommentedBinary(t *testing.T){if got:=Run();got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)}}`, want, want))
}
