package transpiler

import "testing"

func TestCampaignSourceTextCanonicalObjectAnonymousJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>text</groupId><artifactId>object-anonymous</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
public class Main {
 static int calls;
 public static void main(String[] args){
  Object value=new Object(){
   public String String(){calls++;return "ordinary";}
   public String toString(){return "real";}
  };
  System.out.println(String.valueOf(value)+":"+calls);
  Object inherited=new Object(){
   public String String(){calls++;return "ordinary-default";}
   public int hashCode(){return 42;}
  };
  String text=String.valueOf(inherited);
  System.out.println(text.startsWith("probe.Main$")+":"+text.endsWith("@2a")+":"+calls);
 }
}
`,
	}, "probe.Main", "real:0\ntrue:true:0\n")
}
