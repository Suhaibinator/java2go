package transpiler

import "testing"

func TestCampaignSourceTextProtocolCollisionsJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>text</groupId><artifactId>protocol</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
interface Face { String String(); String Java2goToStringExecution(); String toString(); }
class Both implements Face {
 public String String="field";public int Java2goToStringExecution=9;
 int calls;boolean held;
 public String String(){return "ordinary";}
 public String Java2goToStringExecution(){return "hook";}
 public String toString(){calls++;held=Thread.holdsLock(this);return "actual:"+calls;}
}
class Inherited extends Both { }
enum Mode { PLAIN; public String String(){return "enum-ordinary";} }
class Cause extends RuntimeException {
 Cause(){super("detail");}
 public String String(){return "cause-ordinary";}
}
public class Main {public static void main(String[] args){
 Inherited value=new Inherited();Face face=value;
 synchronized(value){System.out.println(String.valueOf((Object)face)+":"+value.held+":"+value.calls);}
 System.out.println(face.String()+":"+face.Java2goToStringExecution()+":"+value.String+":"+value.Java2goToStringExecution);
 class Local implements Face {
  public String String(){return "local-ordinary";}
  public String Java2goToStringExecution(){return "local-hook";}
  public String toString(){return "local";}
 }
 Face local=new Local();Face anonymous=new Face(){
  public String String(){return "anon-ordinary";}
  public String Java2goToStringExecution(){return "anon-hook";}
  public String toString(){return "anon";}
 };
 System.out.println(String.valueOf((Object)local)+":"+local.String()+":"+local.Java2goToStringExecution());
 System.out.println(String.valueOf((Object)anonymous)+":"+anonymous.String()+":"+anonymous.Java2goToStringExecution());
 System.out.println(String.valueOf((Object)Mode.PLAIN)+":"+Mode.PLAIN.String());
 Cause cause=new Cause();System.out.println(String.valueOf((Object)cause)+":"+cause.String());
}}
`,
	}, "probe.Main", "actual:1:true:1\nordinary:hook:field:9\nlocal:local-ordinary:local-hook\nanon:anon-ordinary:anon-hook\nPLAIN:enum-ordinary\nprobe.Cause: detail:cause-ordinary\n")
}
