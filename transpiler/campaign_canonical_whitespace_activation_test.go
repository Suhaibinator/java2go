package transpiler

import "testing"

func TestCampaignCanonicalWhitespaceContentJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>whitespace-content</artifactId><version>1</version></project>`,
		"src/main/java/whiteprobe/Entry.java": `package whiteprobe;
public class Entry {
 public static void main(String[] args) {
  String input = new String(new char[]{' ',(char)0x2003,'X',(char)0xd800,(char)0x2003,' '});
  String trimmed = input.trim();
  String stripped = input.strip();
  System.out.println("units=" + trimmed.length() + ":" + (int)trimmed.charAt(0) + ":" + (int)trimmed.charAt(2) + ":" + stripped.length() + ":" + (int)stripped.charAt(1));
  System.out.println("source=" + input.length() + ":" + (int)input.charAt(3) + ":" + input.isBlank());
  String spaces = "\t\u2003 \n";
  System.out.println("spaces=" + spaces.isBlank() + ":" + spaces.trim().length() + ":" + (spaces.strip() == ""));
  String nonbreaking = new String(new char[]{(char)0xa0,(char)0x2007,(char)0x202f});
  System.out.println("nonbreaking=" + nonbreaking.isBlank() + ":" + (nonbreaking.trim() == nonbreaking) + ":" + (nonbreaking.strip() == nonbreaking));
  String nul = new String(new char[]{0,'X',0});
  System.out.println("nul=" + nul.isBlank() + ":" + nul.trim().equals("X") + ":" + (nul.strip() == nul));
 }
}`,
	}, "whiteprobe.Entry", "units=4:8195:55296:2:55296\nsource=6:55296:false\nspaces=true:1:true\nnonbreaking=false:true:true\nnul=false:true:true\n")
}

func TestCampaignCanonicalWhitespaceIdentityNullJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>whitespace-identity</artifactId><version>1</version></project>`,
		"src/main/java/whiteprobe/Entry.java": `package whiteprobe;
public class Entry {
 static int calls;
 static String absent() { calls++; return null; }
 public static void main(String[] args) {
  String empty = new String(new char[0]);
  System.out.println("empty=" + (empty.trim() == empty) + ":" + (empty.strip() == empty) + ":" + (empty.strip() == "") + ":" + empty.isBlank());
  String plain = new String("abc");
  String padded = new String(" abc ");
  String a = padded.trim(), b = padded.strip();
  System.out.println("identity=" + (plain.trim() == plain) + ":" + (plain.strip() == plain) + ":" + (a == b) + ":" + (a == padded.trim()) + ":" + (b == padded.strip()) + ":" + a.equals(b));
  try { absent().trim(); System.out.println("trim=missed"); } catch(NullPointerException expected) { System.out.println("trim-null=" + calls); }
  try { absent().strip(); System.out.println("strip=missed"); } catch(NullPointerException expected) { System.out.println("strip-null=" + calls); }
  try { absent().isBlank(); System.out.println("blank=missed"); } catch(NullPointerException expected) { System.out.println("blank-null=" + calls); }
 }
}`,
	}, "whiteprobe.Entry", "empty=true:false:true:true\nidentity=true:true:false:false:false:true\ntrim-null=1\nstrip-null=2\nblank-null=3\n")
}

func TestCampaignCanonicalWhitespaceBindingJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>whitespace-binding</artifactId><version>1</version></project>`,
		"src/main/java/owned/String.java": `package owned;
public class String {
 public java.lang.String trim(){return "source-trim";}
 public java.lang.String strip(){return "source-strip";}
 public boolean isBlank(){return false;}
}`,
		"src/main/java/whiteprobe/Entry.java": `package whiteprobe;
import owned.String;
import java.util.function.Function;
import java.util.function.Supplier;
import java.util.function.Predicate;
public class Entry {
 public static void main(java.lang.String[] args) {
  String source = new String();
  System.out.println("source=" + source.trim() + ":" + source.strip() + ":" + source.isBlank());
  Function<java.lang.String,java.lang.String> trim = java.lang.String::trim;
  Function<java.lang.String,java.lang.String> strip = java.lang.String::strip;
  Predicate<java.lang.String> blank = java.lang.String::isBlank;
  System.out.println("refs=" + trim.apply(" x ").equals("x") + ":" + strip.apply("\u2003x\u2003").equals("x") + ":" + blank.test("\u2003"));
  java.lang.String value = new java.lang.String(" old ");
  Supplier<java.lang.String> captured = value::strip;
  value = "new";
  System.out.println("captured=" + captured.get().equals("old"));
  java.lang.String missing = null;
  try { Supplier<java.lang.String> invalid = missing::strip; System.out.println("bound-null=missed"); }
  catch (NullPointerException expected) { System.out.println("bound-null=now"); }
  try { trim.apply(null); System.out.println("unbound-null=missed"); }
  catch (NullPointerException expected) { System.out.println("unbound-null=call"); }
 }
}`,
	}, "whiteprobe.Entry", "source=source-trim:source-strip:false\nrefs=true:true:true\ncaptured=true\nbound-null=now\nunbound-null=call\n")
}
