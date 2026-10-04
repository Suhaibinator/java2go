package transpiler

import "testing"

func TestEnumCloneCanonicalBodyStrictJVM(t *testing.T) {
	files := map[string]string{"pom.xml": `<project><groupId>enumclone</groupId><artifactId>enumclone</artifactId><version>1</version></project>`, "src/main/java/enumclone/Main.java": `package enumclone;
enum Plain {
 ONE;
 Object copy() throws CloneNotSupportedException { return super.clone(); }
}
enum Marked implements java.lang.Cloneable {
 ONE;
 Object copy() throws CloneNotSupportedException { return super.clone(); }
}
enum Body implements java.lang.Cloneable {
 ONE { Object copy() throws CloneNotSupportedException { return super.clone(); } };
 abstract Object copy() throws CloneNotSupportedException;
}
class Enum {
 static int calls;
 protected Object clone() { calls++; return this; }
}
class Child extends Enum {
 Object copy() { return super.clone(); }
}
public class Main {
 public static void main(String[] args) {
  try { Plain.ONE.copy(); } catch(CloneNotSupportedException e) { System.out.println(e.getClass().getName()+":"+e.getMessage()); }
  try { Marked.ONE.copy(); } catch(CloneNotSupportedException e) { System.out.println(e.getClass().getName()+":"+e.getMessage()); }
  try { Body.ONE.copy(); } catch(CloneNotSupportedException e) { System.out.println(e.getClass().getName()+":"+e.getMessage()); }
  Child child=new Child(); System.out.println((child.copy()==child)+":"+Enum.calls);
 }
}`}
	runCampaignCompilerStrictProjectOracle47Args(t, files, "enumclone.Main", "java.lang.CloneNotSupportedException:null\njava.lang.CloneNotSupportedException:null\njava.lang.CloneNotSupportedException:null\ntrue:1\n")
}
