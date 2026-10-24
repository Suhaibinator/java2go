package transpiler

import "testing"

// Canonical catches exercise inherited getMessage without requiring named Go
// constructors or parameter types for reflection-owned exceptions.
func TestCampaignReflectiveThrowableMessageCanonicalJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>reflective-message</artifactId><version>1</version></project>`,
		"src/main/java/reflectionmessages/Main.java": `package reflectionmessages;
public class Main {
 public final int fixed=7;
 public String fail(){throw new IllegalStateException("source failure");}
 public static void main(String[] args) throws Exception {
  Main target=new Main();
  try {Class.forName("missing.ReflectiveMessageProbe");System.out.println("wrong-class");}
  catch(java.lang.ClassNotFoundException e){System.out.println("ClassNotFoundException:"+(e.getMessage()==e.getMessage()));}
  try {Class.forName("missing.ReflectiveMessageProbe");System.out.println("wrong-super");}
  catch(java.lang.ReflectiveOperationException e){System.out.println("ReflectiveOperationException:"+(e.getMessage()==e.getMessage()));}
  try {Main.class.getMethod("missingMethod");System.out.println("wrong-method");}
  catch(java.lang.NoSuchMethodException e){System.out.println("NoSuchMethodException:"+(e.getMessage()==e.getMessage()));}
  try {Main.class.getField("missingField");System.out.println("wrong-field");}
  catch(java.lang.NoSuchFieldException e){System.out.println("NoSuchFieldException:"+(e.getMessage()==e.getMessage()));}
  try {Main.class.getField("fixed").set(target, Integer.valueOf(8));System.out.println("wrong-access");}
  catch(java.lang.IllegalAccessException e){System.out.println("IllegalAccessException:"+(e.getMessage()==e.getMessage()));}
  try {Main.class.getMethod("fail").invoke(target);System.out.println("wrong-invocation");}
  catch(java.lang.reflect.InvocationTargetException e){System.out.println("InvocationTargetException:"+(e.getMessage()==e.getMessage()));}
 }
}
`,
	}, "reflectionmessages.Main", "ClassNotFoundException:true\nReflectiveOperationException:true\nNoSuchMethodException:true\nNoSuchFieldException:true\nIllegalAccessException:true\nInvocationTargetException:true\n")
}

func TestCampaignReflectiveThrowableMessageSourceShadowsJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>review</groupId><artifactId>reflective-message</artifactId><version>1</version></project>`,
		"src/main/java/shadowmessages/Main.java": `package shadowmessages;
import shadow.ReflectiveOperationException;
import shadow.ClassNotFoundException;
import shadow.NoSuchMethodException;
import shadow.NoSuchFieldException;
import shadow.IllegalAccessException;
import shadow.InvocationTargetException;
public class Main {
 public static void main(String[] args){
  System.out.println(new ReflectiveOperationException().getMessage()+":"+new shadow.ReflectiveOperationException().getMessage());
  System.out.println(new ClassNotFoundException().getMessage()+":"+new shadow.ClassNotFoundException().getMessage());
  System.out.println(new NoSuchMethodException().getMessage()+":"+new shadow.NoSuchMethodException().getMessage());
  System.out.println(new NoSuchFieldException().getMessage()+":"+new shadow.NoSuchFieldException().getMessage());
  System.out.println(new IllegalAccessException().getMessage()+":"+new shadow.IllegalAccessException().getMessage());
  System.out.println(new InvocationTargetException().getMessage()+":"+new shadow.InvocationTargetException().getMessage());
 }
}
`,
		"src/main/java/shadow/ReflectiveOperationException.java": `package shadow; public class ReflectiveOperationException {public String getMessage(){return "source-ReflectiveOperationException";}}
`,
		"src/main/java/shadow/ClassNotFoundException.java": `package shadow; public class ClassNotFoundException {public String getMessage(){return "source-ClassNotFoundException";}}
`,
		"src/main/java/shadow/NoSuchMethodException.java": `package shadow; public class NoSuchMethodException {public String getMessage(){return "source-NoSuchMethodException";}}
`,
		"src/main/java/shadow/NoSuchFieldException.java": `package shadow; public class NoSuchFieldException {public String getMessage(){return "source-NoSuchFieldException";}}
`,
		"src/main/java/shadow/IllegalAccessException.java": `package shadow; public class IllegalAccessException {public String getMessage(){return "source-IllegalAccessException";}}
`,
		"src/main/java/shadow/InvocationTargetException.java": `package shadow; public class InvocationTargetException {public String getMessage(){return "source-InvocationTargetException";}}
`,
	}, "shadowmessages.Main", "source-ReflectiveOperationException:source-ReflectiveOperationException\nsource-ClassNotFoundException:source-ClassNotFoundException\nsource-NoSuchMethodException:source-NoSuchMethodException\nsource-NoSuchFieldException:source-NoSuchFieldException\nsource-IllegalAccessException:source-IllegalAccessException\nsource-InvocationTargetException:source-InvocationTargetException\n")
}
