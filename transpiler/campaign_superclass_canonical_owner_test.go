package transpiler

import "testing"

func TestCampaignSuperclassCanonicalOwnerJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>superclass-owner</artifactId><version>1</version></project>`,
		"src/main/java/shadow/NullPointerException.java": `package shadow;
public class NullPointerException extends RuntimeException {
 public NullPointerException(){super("source");}
 public int marker(){return 7;}
}`,
		"src/main/java/probe/Main.java": `package probe;
import shadow.NullPointerException;
public class Main {
 static class QualifiedSource extends shadow.NullPointerException {}
 static class ImportedSource extends NullPointerException {}
 static class QualifiedBuiltin extends java.lang.NullPointerException {
  QualifiedBuiltin(){super("builtin");}
 }
 public static void main(String[] args){
  QualifiedSource source=new QualifiedSource();
  ImportedSource imported=new ImportedSource();
  QualifiedBuiltin builtin=new QualifiedBuiltin();
  System.out.println(source.marker()+":"+imported.marker()+":"+builtin.getMessage());
  System.out.println(NullPointerException.class.isAssignableFrom(builtin.getClass())+":"
   +java.lang.NullPointerException.class.isAssignableFrom(source.getClass())+":"
   +NullPointerException.class.isAssignableFrom(imported.getClass())+":"
   +java.lang.NullPointerException.class.isAssignableFrom(builtin.getClass()));
 }
}`}, "probe.Main", "7:7:builtin\nfalse:false:true:true\n")
}
