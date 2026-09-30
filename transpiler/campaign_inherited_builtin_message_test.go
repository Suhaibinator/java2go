package transpiler

import "testing"

func TestCampaignInheritedBuiltinMessageJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>inherited-message</artifactId><version>1</version></project>`,
		"src/main/java/review/messages/Main.java": `package review.messages;
public class Main {
 static class Failure extends RuntimeException {
  Failure(String text){super(text);}
  @Override public String toString(){return "source:"+getMessage();}
  int length(){return getMessage().length();}
 }
 static class Overloaded extends Failure {
  Overloaded(){super("overload");}
  String getMessage(int index){return "local:"+index;}
  String observed(){return getMessage()+":"+getMessage(2);}
 }
 static class OverridingFailure extends Failure {
  OverridingFailure(){super("ignored");}
  @Override public String getMessage(){return "override";}
 }
 static String getMessage(){return "outer-static";}
 static class Lexical extends RuntimeException {
  Lexical(){super("inherited");}
  String observed(){return getMessage();}
 }
 public static void main(String[] args){
  Failure plain=new Failure("hello"); Failure overridden=new OverridingFailure();
  System.out.println(plain.toString()+":"+plain.length());
  System.out.println(new Overloaded().observed());
  System.out.println(overridden.toString()+":"+overridden.length());
  System.out.println(new Lexical().observed());
 }
}
`}, "review.messages.Main", "source:hello:5\noverload:local:2\nsource:override:8\ninherited\n")
}
