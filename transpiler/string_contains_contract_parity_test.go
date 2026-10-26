package transpiler

import "testing"

const stringContainsContractSource = `public class StringContainsContract {
 static Object lock=new Object();static Thread owner;static String trace="";static RuntimeException failure=new IllegalStateException("exact");
 static class Sequence implements CharSequence {
  int mode;Sequence(int mode){this.mode=mode;}
  public int length(){throw new AssertionError("length");}
  public char charAt(int index){throw new AssertionError("charAt");}
  public CharSequence subSequence(int start,int end){throw new AssertionError("subSequence");}
  public String toString(){trace+="T";if(Thread.currentThread()!=owner||!Thread.holdsLock(lock))throw new AssertionError("execution");if(mode==1)return null;if(mode==2)throw failure;return new String(new char[]{(char)0xd800});}
 }
 static String receiver(boolean missing){trace+="R";return missing?null:new String(new char[]{'a',(char)0xd800,0,(char)0xdfff,'b'});}
 static CharSequence argument(int mode){trace+="A";return new Sequence(mode);}
 public static String run(){synchronized(lock){owner=Thread.currentThread();String out="";
  trace="";boolean present=receiver(false).contains(argument(0));out+=present+":"+trace+"|";
  trace="";try{receiver(true).contains(argument(0));out+="missing";}catch(NullPointerException ex){out+=trace;}out+="|";
  trace="";try{receiver(false).contains(argument(1));out+="missing";}catch(NullPointerException ex){out+=trace;}out+="|";
  trace="";try{receiver(false).contains(argument(2));out+="missing";}catch(RuntimeException ex){out+=(ex==failure)+":"+trace;}out+="|";
  String text=new String(new char[]{'a',(char)0xd800,0,(char)0xdfff,'b'});
  out+=text.contains(new String(new char[]{(char)0xd800,0}))+":"+text.contains(new String(new char[]{(char)0xfffd}))+":"+text.contains("");
  return out;
 }}
 public static void main(String[] args){System.out.print(run());}
}`

func TestStringContainsContractStrictJDK21Parity(t *testing.T) {
	want := stringCaseReferenceJavaOracle(t, "StringContainsContract", stringContainsContractSource)
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>contains-contract</artifactId><version>1</version></project>`,
		"src/main/java/StringContainsContract.java": stringContainsContractSource,
	}, "StringContainsContract", want)
}

const stringContainsShadowSource = `class String {boolean contains(CharSequence value){return false;}}
public class StringContainsShadow {
 static <String extends CharSequence> boolean binder(String value){return "xx".contains(value);}
 public static java.lang.String run(){String source=new String();return source.contains("x")+":"+new java.lang.String("xx").contains("x")+":"+binder("x");}
 public static void main(java.lang.String[] args){System.out.print(run());}
}`

func TestStringContainsShadowsStrictJDK21Parity(t *testing.T) {
	want := stringCaseReferenceJavaOracle(t, "StringContainsShadow", stringContainsShadowSource)
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>contains-shadows</artifactId><version>1</version></project>`,
		"src/main/java/StringContainsShadow.java": stringContainsShadowSource,
	}, "StringContainsShadow", want)
}
