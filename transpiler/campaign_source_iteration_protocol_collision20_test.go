package transpiler

import "testing"

func TestCampaignSourceIterationProtocolCollisionJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>iteration</groupId><artifactId>collision</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
import java.util.Iterator;
interface Face extends Iterator<String> {
 String NextJava2goExecution();
 String NextJava2goExecution0();
 String NextJava2goIterationBodyExecution();
}
class Cursor implements Face {
 int used;
 public boolean hasNext(){return used==0;}
 public String next(){used++;return "next";}
 public String NextJava2goExecution(){return "ordinary";}
 public String NextJava2goExecution0(){return "zero";}
 public String NextJava2goIterationBodyExecution(){return "body";}
}
public class Main {
 static String read(Face value){Iterator<String> view=value;return view.next()+":"+value.NextJava2goExecution()+":"+value.NextJava2goExecution0()+":"+value.NextJava2goIterationBodyExecution()+":"+view.hasNext()+":"+(view==value);}
 public static void main(String[] args){
  System.out.println(read(new Cursor()));
  class Local extends Cursor { public int IteratorJava2goExecution=7; }
  Local local=new Local();System.out.println(read(local)+":"+local.IteratorJava2goExecution);
  Face anonymous=new Face(){
   int used;
   public boolean hasNext(){return used==0;}
   public String next(){used++;return "next";}
   public String NextJava2goExecution(){return "ordinary";}
   public String NextJava2goExecution0(){return "zero";}
   public String NextJava2goIterationBodyExecution(){return "body";}
  };
  System.out.println(read(anonymous));
 }
}
`,
	}, "probe.Main", "next:ordinary:zero:body:false:true\nnext:ordinary:zero:body:false:true:7\nnext:ordinary:zero:body:false:true\n")
}
