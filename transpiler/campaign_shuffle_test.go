package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignSeededShuffleJVMParity(t *testing.T) {
	const source = `import java.util.ArrayList;
 import java.util.Collections;
 import java.util.List;
 import java.util.Random;
 public class ShuffleWorkflow {
  public static String run(){
   Random random=new Random(41);
   String output="";
   for(int size=0;size<12;size++) {
    List<Integer> list=new ArrayList<>();
    for(int i=0;i<size;i++){list.add(i);}
    Collections.shuffle(list,random);
    output+=list.toString()+":"+random.nextInt()+";";
   }
   return output;
  }
 }`
	want := campaignRuntimeJavaOracle(t, "ShuffleWorkflow", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
 import (
  "slices"
  "testing"
  "unicode/utf16"
 )
 func TestShuffle(t *testing.T){if got:=Run();got==nil||!slices.Equal(got.UTF16Copy(),utf16.Encode([]rune(%q))){t.Fatalf("got %%v want %%q",got,%q)}}`, want, want))
}
