package transpiler

import "testing"

// A separate API exercises the same Java-int constant capture boundary as the
// prefix offset fixture. Expected observations are validated on actual JDK21.
func TestCampaignCanonicalPrimitiveArgumentCaptureJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>primitive-capture</artifactId><version>1</version></project>`,
		"src/main/java/captureprobe/Entry.java": `package captureprobe;
public class Entry {
 static int effects;
 static String absent() { effects = effects * 10 + 1; return null; }
 static int index() { effects = effects * 10 + 2; return 0; }
 public static void main(String[] args) {
  String text = new String("abc");
  System.out.println("literal=" + text.charAt(1));
  try { text.charAt(Integer.MAX_VALUE); System.out.println("max=missed"); } catch(StringIndexOutOfBoundsException expected) { System.out.println("max=" + expected.getMessage()); }
  try { text.charAt(Integer.MIN_VALUE); System.out.println("min=missed"); } catch(StringIndexOutOfBoundsException expected) { System.out.println("min=" + expected.getMessage()); }
  try { absent().charAt(index()); System.out.println("null=missed"); } catch(NullPointerException expected) { System.out.println("null=" + effects); }
 }
}`,
	}, "captureprobe.Entry", "literal=b\nmax=Index 2147483647 out of bounds for length 3\nmin=Index -2147483648 out of bounds for length 3\nnull=12\n")
}
