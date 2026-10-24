package transpiler

import "testing"

func TestCampaignInheritedBuiltinMessageVarargsPrecedenceJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>message-varargs-precedence</artifactId><version>1</version></project>`,
		"src/main/java/review/messagevarargs/Main.java": `package review.messagevarargs;
public class Main {
 static String trace="";
 static String mark(String value){trace+="A;";return value;}
 static class Failure extends RuntimeException {
  Failure(){super("base");}
  public String getMessage(String... values){trace+="V"+values.length+";";return "var:"+values.length;}
  String exercise(){
   String a=getMessage();
   String b=getMessage(mark("x"));
   String c=this.getMessage();
   String d=this.getMessage(mark("x"),mark("y"));
   return a+"|"+b+"|"+c+"|"+d;
  }
 }
 public static void main(String[] args){
  Failure failure=new Failure();
  System.out.println(failure.exercise());
  System.out.println(failure.getMessage()+":"+failure.getMessage(mark("z")));
  System.out.println(trace);
 }
}
`}, "review.messagevarargs.Main", "base|var:1|base|var:2\nbase:var:1\nA;V1;A;A;V2;A;V1;\n")
}
