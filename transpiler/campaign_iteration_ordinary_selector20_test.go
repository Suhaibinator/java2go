package transpiler

import "testing"

func TestCampaignOrdinaryIterationSelectorNamesJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>ordinary</groupId><artifactId>selectors</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
interface Face {
 String NextJava2goExecution();
 String NextJava2goExecution0();
}
class Named implements Face {
 public String NextJava2goExecution(){return "named";}
 public String NextJava2goExecution0(){return "zero";}
}
public class Main {
 static String read(Face value){return value.NextJava2goExecution()+":"+value.NextJava2goExecution0();}
 public static void main(String[] args){
  System.out.println(read(new Named()));
  class Local implements Face {
   public int IteratorJava2goExecution=5;
   public String NextJava2goExecution(){return "local";}
   public String NextJava2goExecution0(){return "zero";}
  }
  Local local=new Local();System.out.println(read(local)+":"+local.IteratorJava2goExecution);
  Face anonymous=new Face(){
   public String NextJava2goExecution(){return "anonymous";}
   public String NextJava2goExecution0(){return "zero";}
  };
  System.out.println(read(anonymous));
 }
}
`,
	}, "probe.Main", "named:zero\nlocal:zero:5\nanonymous:zero\n")
}
