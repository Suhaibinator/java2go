package transpiler

import "testing"

func TestCampaignEnclosingStaticReturnInferenceJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>enclosing-static-return</artifactId><version>1</version></project>`,
		"src/main/java/review/enclosingreturn/Main.java": `package review.enclosingreturn;
public class Main {
 static int pick(String value){return value.length()+3;}
 static long pick(int value){return value+8L;}
 static <T> T echo(T value){return value;}
 static class Worker{
  Integer first(){return pick("four");}
  Long second(){return pick(1);}
  String third(){return echo("value");}
 }
 public static void main(String[] args){Worker worker=new Worker();System.out.println(worker.first()+":"+worker.second()+":"+worker.third());}
}
`}, "review.enclosingreturn.Main", "7:9:value\n")
}
