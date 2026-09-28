package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignSeededRandomJVMParity(t *testing.T) {
	const source = `import java.util.Random;
 import java.util.Arrays;
 public class SeededRandom {
  public static String run() {
   String output="";
   for(long seed:new long[]{0L,-1L,17L,9223372036854775807L}) {
    Random random=new Random(seed);
    for(int i=0;i<12;i++) {
     output+=random.nextInt()+":"+random.nextInt(1073741825)+":"+random.nextInt(16)+":"+random.nextBoolean()+":"+random.nextLong()+":"+(int)(random.nextFloat()*16777216.0f)+":"+(long)(random.nextDouble()*9007199254740992.0)+";";
    }
    byte[] bytes=new byte[7];random.nextBytes(bytes);output+=Arrays.toString(bytes);
    random.setSeed(seed);int first=random.nextInt();random.setSeed(seed);output+=":"+(first==random.nextInt());
    try {random.nextInt(0);} catch(IllegalArgumentException expected){output+=":"+expected.getMessage();}
   }
   int intSeed=41;
   Integer boxedSeed=Integer.valueOf(41);
   Random plain=new Random(intSeed);
   Random boxed=new Random(boxedSeed);
   output+=":"+(plain.nextInt()==boxed.nextInt());
   boxed.setSeed(intSeed);plain.setSeed(boxedSeed);
   output+=":"+(plain.nextLong()==boxed.nextLong());
   return output;
  }
 }`
	want := campaignRuntimeJavaOracle(t, "SeededRandom", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
 import "testing"
 func TestRandom(t *testing.T){if got:=Run();got!=%q{t.Fatalf("got %%q want %%q",got,%q)}}`, want, want))
}
