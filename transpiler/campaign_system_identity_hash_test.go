package transpiler

import "testing"

func TestCampaignSystemIdentityHashCodeJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>identityprobe</groupId><artifactId>probe</artifactId><version>1</version></project>`,
		"src/main/java/identityprobe/System.java": `package identityprobe;
public class System { public static int identityHashCode(Object value){return -13;} }`,
		"src/main/java/identityprobe/Main.java": `package identityprobe;
public class Main {
 static class Base { int calls;public int hashCode(){calls++;return 71;} }
 static class Child extends Base {}
 static class Binder { public int identityHashCode(Object value){return 42;} }
 static int shadow(Binder System){return System.identityHashCode(null);}
 public static void main(String[] args){
  Child child=new Child();Base base=child;Object erased=child;
  int first=java.lang.System.identityHashCode(child),second=java.lang.System.identityHashCode(base),third=java.lang.System.identityHashCode(erased);
  java.lang.System.out.println((first==second)+":"+(second==third)+":"+child.calls);
  java.lang.System.out.println(java.lang.System.identityHashCode(null)+":"+java.lang.System.identityHashCode((Object)null));
  int[] array={1};Object alias=array;String expected="[I@"+Integer.toHexString(java.lang.System.identityHashCode(alias));
  java.lang.System.out.println(String.valueOf(array).equals(expected)+":"+(java.lang.System.identityHashCode(array)==array.hashCode()));
  java.lang.System.out.println(System.identityHashCode(child)+":"+shadow(new Binder()));
  java.lang.System.out.println(child.hashCode()+":"+child.calls);
  Integer boxed=7;java.lang.System.out.println(java.lang.System.identityHashCode(boxed)==java.lang.System.identityHashCode((Object)boxed));
 }
}`,
	}, "identityprobe.Main", "true:true:0\n0:0\ntrue:true\n-13:42\n71:1\ntrue\n")
}
