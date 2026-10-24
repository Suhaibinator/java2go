package transpiler

import "testing"

func TestObjectCloneReservedProtocolSourceInterfaceStrictJVM(t *testing.T) {
	files := map[string]string{"pom.xml": `<project><groupId>clone</groupId><artifactId>protocol</artifactId><version>1</version></project>`, "src/main/java/collision/Main.java": `package collision;
public class Main {
 interface Callback { Object Java2goCloneSubobject(Object ignored); }
 static class Value implements java.lang.Cloneable,Callback {
  public Object Java2goCloneSubobject(Object ignored) { return this; }
  Value copy() throws CloneNotSupportedException { return (Value)super.clone(); }
 }
 public static void main(String[] args) throws Exception {
  Value original=new Value(); Value copy=original.copy();
  Callback callback=copy;
  System.out.println((copy!=original)+":"+(callback.Java2goCloneSubobject(null)==copy));
 }
}`}
	runCampaignCompilerStrictProjectOracle47Args(t, files, "collision.Main", "true:true\n")
}
