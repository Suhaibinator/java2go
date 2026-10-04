package transpiler

import "testing"

func TestCampaignThrowableCauseDeclaredReferenceJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>throwablecause</groupId><artifactId>projection</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
class SourceCause extends Exception {SourceCause(){super("source");}}
public class Main {
 static int reads;static Exception stored;
 static Exception receiver(){reads++;return stored;}
 static Throwable returnCause(Exception value){return value.getCause();}
 static boolean same(Throwable left,Throwable right){return left==right;}
 public static void main(String[] args){
  SourceCause source=new SourceCause();stored=new Exception("outer",source);
  Throwable direct=source;Throwable from=receiver().getCause();Throwable empty=new Exception().getCause();
  Object erased=from;Throwable cast=(Throwable)erased;Throwable nil=(Throwable)(Object)null;
  Throwable later=null;later=stored.getCause();
  System.out.println((direct==source)+":"+(from==source)+":"+(empty==null)+":"+(cast==source)+":"+(nil==null)+":"+(later==source)+":"+same(returnCause(stored),source)+":"+reads);
 }
}`,
	}, "probe.Main", "true:true:true:true:true:true:true:1\n")
}
