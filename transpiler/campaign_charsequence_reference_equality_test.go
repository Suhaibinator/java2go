package transpiler

import "testing"

func TestCampaignCharSequenceReferenceEqualityJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>charsequence-identity</artifactId><version>1</version></project>`,
		"src/main/java/p/Main.java": `package p;public class Main {
 static class Text implements java.lang.CharSequence {
  public int length(){return 1;}public char charAt(int index){return 'x';}
  public java.lang.CharSequence subSequence(int start,int end){return this;}
 }
 static Text absent(){return null;}
 public static void main(String[]args){
  java.lang.CharSequence missing=absent();
  System.out.println((missing==null)+":"+(null==missing)+":"+(missing!=null)+":"+(null!=missing));
  Text source=new Text();java.lang.CharSequence alias=source;java.lang.CharSequence other=new Text();
  System.out.println((alias==source)+":"+(source==alias)+":"+(alias!=source)+":"+(source!=alias));
  System.out.println((alias==other)+":"+(other==alias)+":"+(alias!=other)+":"+(other!=alias));
 }
}`,
	}, "p.Main", "true:true:false:false\ntrue:true:false:false\nfalse:false:true:true\n")
}
