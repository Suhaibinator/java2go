package transpiler

import "testing"

func TestCampaignSourceTextReservedAliasFamiliesJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>text</groupId><artifactId>aliases</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
interface Face {
 String String(); String String0();
 String Java2goToStringExecution(); String Java2goToStringExecution0();
 String toString();
}
class Named implements Face {
 public String String(){return "named-String";}
 public String String0(){return "named-String0";}
 public String Java2goToStringExecution(){return "named-hook";}
 public String Java2goToStringExecution0(){return "named-hook0";}
 public String toString(){return "named-text";}
}
class Child extends Named { }
public class Main {
 static void show(Face value){
  System.out.println(value.String()+":"+value.String0()+":"+value.Java2goToStringExecution()+":"+value.Java2goToStringExecution0()+":"+String.valueOf((Object)value));
 }
 public static void main(String[] args){
  show(new Child());
  class Local implements Face {
   public String String(){return "local-String";}
   public String String0(){return "local-String0";}
   public String Java2goToStringExecution(){return "local-hook";}
   public String Java2goToStringExecution0(){return "local-hook0";}
   public String toString(){return "local-text";}
  }
  show(new Local());
  show(new Face(){
   public String String(){return "anon-String";}
   public String String0(){return "anon-String0";}
   public String Java2goToStringExecution(){return "anon-hook";}
   public String Java2goToStringExecution0(){return "anon-hook0";}
   public String toString(){return "anon-text";}
  });
 }
}
`,
	}, "probe.Main", "named-String:named-String0:named-hook:named-hook0:named-text\nlocal-String:local-String0:local-hook:local-hook0:local-text\nanon-String:anon-String0:anon-hook:anon-hook0:anon-text\n")
}

func TestCampaignSourceTextProtocolEmbeddedNamesJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>text</groupId><artifactId>embeds</artifactId><version>1</version></project>`,
		"src/main/java/marker/Java2goToStringExecution.java": `package marker; public interface Java2goToStringExecution { }`,
		"src/main/java/base/Java2goToStringExecution.java":   `package base; public class Java2goToStringExecution { public String toString(){return "base";} }`,
		"src/main/java/probe/Main.java": `package probe;
class Marked implements marker.Java2goToStringExecution {
 public String toString(){return "marked";}
}
class Child extends base.Java2goToStringExecution {
 public String toString(){return "child";}
}
public class Main {public static void main(String[] args){
 Marked marked=new Marked();Child child=new Child();
 System.out.println(String.valueOf((Object)marked)+":"+marked.toString());
 System.out.println(String.valueOf((Object)child)+":"+child.toString());
}}
`,
	}, "probe.Main", "marked:marked\nchild:child\n")
}
