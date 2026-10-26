package transpiler

import "testing"

func TestCampaignInheritedBuiltinMessageExecutionJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>inherited-message-execution</artifactId><version>1</version></project>`,
		"src/main/java/review/messagesexecution/Main.java": `package review.messagesexecution;
public class Main {
 static Thread expected;
 static int calls;
 static class Base extends RuntimeException {
  Base(){super("base");}
  String observed(){return getMessage();}
 }
 static class Derived extends Base {
  public String getMessage(){calls++;return (Thread.currentThread()==expected)+":"+super.getMessage();}
  String getMessage(int ignored){return "overload";}
  String getMessageJava2goExecution(){return "collision";}
 }
 static class Further extends Derived {
  public String getMessage(){return "further:"+super.getMessage();}
 }
 public static void main(String[] args) throws Exception {
  Base value=new Further();
  Thread worker=new Thread(() -> {
   expected=Thread.currentThread();
   System.out.println(value.observed());
   Throwable erased=value;
   System.out.println(erased.getMessage()+":"+calls);
   Throwable absent=null;
   try {absent.getMessage();System.out.println("bad-null");}
   catch(NullPointerException right){System.out.println("null-checked");}
  });
  worker.start();worker.join();
 }
}
`}, "review.messagesexecution.Main", "further:true:base\nfurther:true:base:2\nnull-checked\n")
}
