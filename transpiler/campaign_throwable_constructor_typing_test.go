package transpiler

import "testing"

func TestCampaignThrowableConstructorBoundStringJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>throwable-bound-string</artifactId><version>1</version></project>`,
		"src/main/java/throwtyping/Entry.java": `package throwtyping;
class Child extends Exception {
 <S extends java.lang.String> Child(S message) { super(message); }
}
public class Entry {
 static <S extends java.lang.String> Exception make(S message) { return new Exception(message); }
 public static void main(String[] args) {
  String text = new String(new char[]{'g',(char)0xd800});
  Exception direct = make(text);
  Child inherited = new Child(text);
  RuntimeException cause = new RuntimeException("cause");
  direct.initCause(cause);
  inherited.initCause(cause);
  System.out.println("generic=" + (direct.getMessage() == text) + ":" + (inherited.getMessage() == text) + ":" + (direct.getCause() == cause) + ":" + (inherited.getCause() == cause) + ":" + (int)direct.getMessage().charAt(1));
  Exception absent = Entry.<String>make(null);
  absent.initCause(cause);
  System.out.println("null=" + (absent.getMessage() == null) + ":" + (absent.getCause() == cause));
 }
}`,
	}, "throwtyping.Entry", "generic=true:true:true:true:55296\nnull=true:true\n")
}

func TestCampaignThrowableConstructorBareNullPairJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>throwable-null-pair</artifactId><version>1</version></project>`,
		"src/main/java/throwtyping/Entry.java": `package throwtyping;
class Child extends Exception { Child(Throwable cause) { super(null, cause); } }
public class Entry {
 public static void main(String[] args) {
  RuntimeException cause = new RuntimeException("cause");
  Exception direct = new Exception(null, cause);
  Child inherited = new Child(cause);
  Exception empty = new Exception(null, null);
  System.out.println("pair=" + (direct.getMessage() == null) + ":" + (direct.getCause() == cause) + ":" + (inherited.getMessage() == null) + ":" + (inherited.getCause() == cause));
  System.out.println("empty=" + (empty.getMessage() == null) + ":" + (empty.getCause() == null));
 }
}`,
	}, "throwtyping.Entry", "pair=true:true:true:true\nempty=true:true\n")
}
