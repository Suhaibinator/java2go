package transpiler

import "testing"

func TestCampaignCapitalStringIsNotToStringJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>text</groupId><artifactId>capital</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
interface Capital { String String(); }
class Plain implements Capital {
 int calls;int hashes;boolean held;
 public String String(){calls++;return "ordinary";}
 public String String(int ignored){calls++;return "overload";}
 public int hashCode(){hashes++;held=Thread.holdsLock(this);return 42;}
}
class Child extends Plain { }
class Real {
 int calls;boolean held;
 public String toString(){calls++;held=Thread.holdsLock(this);return "real:"+calls;}
}
class RealChild extends Real { }
public class Main {public static void main(String[] args){
 Child child=new Child();Object object=child;Capital view=child;
 synchronized(child){
  String direct=String.valueOf(child);String erased=String.valueOf(object);String concat="["+object+"]";
  System.out.println(direct.equals("probe.Child@2a")+":"+erased.equals(direct)+":"+concat.equals("["+direct+"]")+":"+child.calls+":"+child.hashes+":"+child.held);
 }
 System.out.println(view.String()+":"+child.String(1)+":"+child.calls);
 RealChild real=new RealChild();Object erasedReal=real;
 synchronized(real){System.out.println(String.valueOf(erasedReal)+":"+real.toString()+":"+real.held);}
}}
`,
	}, "probe.Main", "true:true:true:0:3:true\nordinary:overload:2\nreal:1:real:2:true\n")
}
