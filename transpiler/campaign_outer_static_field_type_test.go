package transpiler

import "testing"

func TestCampaignOuterStaticFieldReceiverTypingJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>fieldprobe</groupId><artifactId>fieldprobe</artifactId><version>1</version></project>`,
		"src/main/java/state/Parent.java": `package state;
import java.util.*;
public class Parent {
 protected static List<String> inherited = new ArrayList<>();
 static {inherited.add("parent");}
}`,
		"src/main/java/app/Outer.java": `package app;
import java.util.*;
public class Outer extends state.Parent {
 static List<String> values=new ArrayList<>();
 static {values.add("outer");}
 static class Nested {
  static String apply() {
   values.add("extra");values.set(0,values.get(0).toUpperCase());
   inherited.add("second");
   return values.get(0)+":"+values.size()+":"+inherited.get(0).toUpperCase()+":"+inherited.size();
  }
  static String parameter(String values){return values.toUpperCase();}
  static String local(){String values="local";return values.toUpperCase();}
 }
 static class Holder {List<String> values=new ArrayList<>();Holder(){values.add("holder");}}
 static class Child extends Holder {String apply(){values.add("child");return values.get(0).toUpperCase()+":"+values.size();}}
 class Inner {String apply(){return values.get(0)+":"+inherited.size();}}
 public static String run(){Outer owner=new Outer();return Nested.apply()+"|"+Nested.parameter("parameter")+"|"+Nested.local()+"|"+new Child().apply()+"|"+owner.new Inner().apply();}
}`,
		"src/main/java/app/Main.java": `package app;public class Main {public static void main(String[]args){System.out.println(Outer.run());}}`,
	}, "app.Main", "OUTER:2:PARENT:2|PARAMETER|LOCAL|HOLDER:2|OUTER:2\n")
}
