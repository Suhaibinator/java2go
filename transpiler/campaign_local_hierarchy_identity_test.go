package transpiler

import "testing"

func TestCampaignLocalHierarchyIdentityAndTextJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>identity</groupId><artifactId>local</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
public class Main {
 public static void main(String[] args) {
  class Base {
   int inherited; int observed;
   Base(int value) { inherited=value; observed=probe(); }
   int probe() { return -1; }
   public int hashCode() { return 42; }
  }
  Base plain=new Base(3);
  Base child=new Base(7) { int probe() { return inherited+2; } };
  Object erased=child;
  Base restored=(Base)erased;
  String text=erased.toString();
  System.out.println((child.observed*10+child.inherited)+":"+(restored==child)+":"+
    (erased==restored)+":"+text.equals(child.toString())+":"+
    text.endsWith("@2a")+":"+!text.equals(plain.toString()));
 }
}`,
	}, "probe.Main", "97:true:true:true:true:true\n")
}
