package transpiler

import "testing"

func TestCampaignSingleFileEntryTypeCollisionJVM(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main {
 static String label(){return "entry";}
 public static void main(String[] args){
  java.util.function.Supplier<String> ref=Main::label;
  System.out.println(Main.class.getName()+":"+Main0.class.getName()+":"+ref.get());
 }
}`, "Main:Main0:entry\n", map[string]string{"Main0.java": `public class Main0 {}`})
}

func TestCampaignStaticMethodTypeCollisionProjectJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                                          `<project><groupId>probe</groupId><artifactId>static-types</artifactId><version>1</version></project>`,
		"src/main/java/probe/Token.java":                   `package probe; public class Token {}`,
		"src/main/java/probe/Token0.java":                  `package probe; public class Token0 {}`,
		"src/main/java/probe/Token01Java2goExecution.java": `package probe; public class Token01Java2goExecution {}`,
		"src/main/java/probe/Main.java": `package probe; public class Main {
 public static String Token(){return "called";}
 public static void main(String[] args){
  java.util.function.Supplier<String> ref=Main::Token;
  System.out.println(Token()+":"+ref.get()+":"+Token.class.getName()+":"+Token0.class.getName());
 }
}`}, "probe.Main", "called:called:probe.Token:probe.Token0\n")
}
