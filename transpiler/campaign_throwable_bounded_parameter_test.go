package transpiler

import "testing"

func TestCampaignThrowableBoundedTypeParameterMessageJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>bounded-throwable</artifactId><version>1</version></project>`,
		"src/main/java/p/Main.java": `package p; public class Main {
 static <Exception extends java.lang.RuntimeException> String message(Exception value){return value.getMessage();}
 public static void main(String[] args){System.out.println(message(new RuntimeException("ok")));}
 }`,
	}
	runCampaignCompilerStrictProjectOracle(t, files, "p.Main", "ok\n")
}

func TestCampaignThrowableTypeParameterBoundControlsJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>throwable-bound-controls</artifactId><version>1</version></project>`,
		"src/main/java/p/Main.java": `package p; public class Main {
 static class Holder<Exception extends java.lang.RuntimeException> {
  String read(Exception value){return value.getMessage();}
 }
 static String choose(Object value){return "object";}
 static String choose(Throwable value){return "throwable";}
 static <Throwable extends CharSequence> String control(Throwable value){return choose(value);}
 public static void main(String[]args){
  Holder<RuntimeException> holder=new Holder<RuntimeException>();
  System.out.println(holder.read(new RuntimeException("class")));
  System.out.println(control("text"));
 }
 }`,
	}
	runCampaignCompilerStrictProjectOracle(t, files, "p.Main", "class\nobject\n")
}

func TestCampaignThrowableOuterBoundDeclarationJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>outer-throwable-bound</artifactId><version>1</version></project>`,
		"src/main/java/p/Main.java": `package p; public class Main {
 static class Box<Exception extends RuntimeException>{
  Exception saved;
  Box(Exception value){saved=value;}
  <Exception> String shadowParameter(Exception ignored){return saved.getMessage();}
  <RuntimeException> String shadowBound(RuntimeException ignored){return saved.getMessage();}
 }
 static class Chain<A extends RuntimeException,B extends A>{
  B saved;
  Chain(B value){saved=value;}
  <A> String read(A ignored){return saved.getMessage();}
 }
 public static void main(String[]args){
  Box<RuntimeException> box=new Box<RuntimeException>(new RuntimeException("outer"));
  Chain<RuntimeException,RuntimeException> chain=new Chain<RuntimeException,RuntimeException>(new RuntimeException("chain"));
  System.out.println(box.shadowParameter("text")+":"+box.shadowBound("text")+":"+chain.read("text"));
 }
 }`,
	}
	runCampaignCompilerStrictProjectOracle(t, files, "p.Main", "outer:outer:chain\n")
}
