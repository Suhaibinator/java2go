package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignMessageDigestJVMParity(t *testing.T) {
	const source = `import java.security.MessageDigest;
 import java.security.NoSuchAlgorithmException;
 import java.nio.ByteBuffer;
 import java.util.Arrays;
 public class DigestWorkflow {
  public static String run() throws Exception {
   MessageDigest digest = MessageDigest.getInstance("sHa-256");
   byte[] bytes = new byte[]{97,98,99,100,101};
   digest.update((byte)97);
   digest.update(bytes, 1, 1);
   ByteBuffer buffer = ByteBuffer.wrap(bytes, 2, 1);
   digest.update(buffer);
   String first = Arrays.toString(digest.digest());
   String empty = Arrays.toString(digest.digest());
   digest.update(bytes);
   digest.reset();
   String again = Arrays.toString(digest.digest(new byte[]{97,98,99}));
   boolean missing = false;
   try { MessageDigest.getInstance("NoSuchDigest"); }
   catch (NoSuchAlgorithmException expected) { missing = true; }
   return first + ":" + empty + ":" + first.equals(again) + ":" + buffer.position()
    + ":" + digest.getAlgorithm() + ":" + digest.getDigestLength() + ":" + missing
    + ":" + Arrays.toString(MessageDigest.getInstance("SHA-512/256").digest(bytes))
    + ":" + Arrays.toString(MessageDigest.getInstance("SHA3-256").digest(bytes));
  }
 }`
	want := campaignRuntimeJavaOracle(t, "DigestWorkflow", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
 import "testing"
 func TestDigest(t *testing.T) { if got:=Run();got!=%q {t.Fatalf("got %%q want %%q",got,%q)} }
 `, want, want))
}

func TestCampaignMessageDigestFailedUpdateJVMParity(t *testing.T) {
	const source = `import java.security.MessageDigest;
 import java.util.Arrays;
 public class DigestFailures {
  static String range(int offset,int length) throws Exception {
   MessageDigest digest=MessageDigest.getInstance("SHA-256");
   digest.update((byte)97);
   String result="ok";
   try {digest.update(new byte[]{98,99,100},offset,length);}
   catch(RuntimeException e){result=e.getClass().getSimpleName()+":"+e.getMessage();}
   return result+"|"+Arrays.toString(digest.digest());
  }
  public static String run() throws Exception {
   return range(-1,1)+";"+range(-1,0)+";"+range(3,0)+";"+range(4,0)+";"+range(0,-1)+";"+range(2,2);
  }
 }`
	want := campaignRuntimeJavaOracle(t, "DigestFailures", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
 import "testing"
 func TestDigestFailures(t *testing.T){if got:=Run();got!=%q{t.Fatalf("got %%q want %%q",got,%q)}}`, want, want))
}
