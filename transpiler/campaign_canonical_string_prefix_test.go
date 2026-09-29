package transpiler

import "testing"

// Prepared before implementation; the strict helper validates the handwritten
// observations on explicit JDK21 before transpiling the unchanged Java source.
func TestCampaignCanonicalStringPrefixSuffixJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>string-prefix</artifactId><version>1</version></project>`,
		"src/main/java/prefixprobe/Entry.java": `package prefixprobe;
import java.util.function.BiPredicate;
import java.util.function.Predicate;
public class Entry {
 static int effects;
 static String absent() { effects = effects * 10 + 1; return null; }
 static String argument() { effects = effects * 10 + 2; return "x"; }
 public static void main(String[] args) {
  String value = new String(new char[]{'A',(char)0xd800,'B',(char)0xdc00});
  String high = new String(new char[]{(char)0xd800});
  String low = new String(new char[]{(char)0xdc00});
  System.out.println("units=" + value.startsWith("A") + ":" + value.startsWith(high,1) + ":" + value.endsWith(low) + ":" + value.startsWith(low,1) + ":" + value.endsWith(high));
  System.out.println("offset=" + value.startsWith("",-1) + ":" + value.startsWith("",0) + ":" + value.startsWith("",4) + ":" + value.startsWith("",5) + ":" + value.startsWith("",Integer.MAX_VALUE) + ":" + value.startsWith("",Integer.MIN_VALUE));
  System.out.println("length=" + value.startsWith("ABCDE") + ":" + value.endsWith("ABCDE") + ":" + value.startsWith("B",2) + ":" + value.startsWith("B",3) + ":" + value.endsWith("") + ":" + "".endsWith("") + ":" + "".startsWith("A"));
  System.out.println("prefix-null-negative=" + value.startsWith(null,-1));
  try { value.startsWith(null,0); System.out.println("prefix-null=missed"); } catch(NullPointerException expected) { System.out.println("prefix-null=now"); }
  try { value.endsWith(null); System.out.println("suffix-null=missed"); } catch(NullPointerException expected) { System.out.println("suffix-null=now"); }
  try { absent().startsWith(argument()); System.out.println("receiver-null=missed"); } catch(NullPointerException expected) { System.out.println("receiver-null=" + effects); }
  effects = 0;
  try { absent().endsWith(argument()); System.out.println("end-receiver-null=missed"); } catch(NullPointerException expected) { System.out.println("end-receiver-null=" + effects); }
  BiPredicate<String,String> starts = java.lang.String::startsWith;
  BiPredicate<String,String> ends = java.lang.String::endsWith;
  Predicate<String> captured = value::startsWith;
  value = "Z";
  System.out.println("refs=" + starts.test("abc","a") + ":" + ends.test("abc","c") + ":" + captured.test("A"));
  String missing = null;
  try { Predicate<String> invalid = missing::endsWith; System.out.println("bound-null=missed"); } catch(NullPointerException expected) { System.out.println("bound-null=now"); }
  try { starts.test(null,""); System.out.println("unbound-null=missed"); } catch(NullPointerException expected) { System.out.println("unbound-null=call"); }
 }
}`,
	}, "prefixprobe.Entry", "units=true:true:true:false:false\noffset=false:true:true:false:false:false\nlength=false:false:true:false:true:true:false\nprefix-null-negative=false\nprefix-null=now\nsuffix-null=now\nreceiver-null=12\nend-receiver-null=12\nrefs=true:true:true\nbound-null=now\nunbound-null=call\n")
}
