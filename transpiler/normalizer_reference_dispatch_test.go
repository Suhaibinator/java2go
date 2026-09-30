package transpiler

import (
	"fmt"
	"testing"
)

// Keep the source-owned names and qualified JDK calls in the same compiled
// program so short-name dispatch cannot silently substitute either owner.
func TestCanonicalNormalizerSourceOwnerAndQualifiedDispatch(t *testing.T) {
	const source = `class Normalizer {
 static String normalize(String input,int form){return "source:"+input+":"+form;}
 static boolean isNormalized(String input,int form){return false;}
}
public class CanonicalNormalizerSourceOwner {
 public static String run(){return Normalizer.normalize("e\u0301",7)+":"+Normalizer.isNormalized("e\u0301",7)+":"+java.text.Normalizer.normalize("e\u0301",java.text.Normalizer.Form.NFC)+":"+java.text.Normalizer.isNormalized("e\u0301",java.text.Normalizer.Form.NFC);}
 public static void main(String[] args){System.out.print(run());}
}`
	want := campaignRuntimeJavaOracle(t, "CanonicalNormalizerSourceOwner", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestNormalizerOwners(t *testing.T){if got:=Run();!got.Equals(j.JavaStringFromHostUTF8(%q)){t.Fatal("source and qualified Normalizer owners differ from exact JVM stream")}}
`, want))
}
