package transpiler

import "testing"

func TestCampaignCancellationExceptionReferenceJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>cancellation-overloads</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
import java.util.concurrent.CancellationException;
class Child extends CancellationException {<S extends String> Child(S value){super(value);}}
public class Main {
 static int calls;static String message(){calls++;return new String(new char[]{'r',0,(char)0xd800});}
 static <S extends String> CancellationException bound(S value){return new CancellationException(value);}
 public static void main(String[] args){
 CancellationException empty=new CancellationException();CancellationException absent=new CancellationException(null);String text=message();CancellationException value=new CancellationException(text);CancellationException other=new CancellationException(text);
 System.out.println((empty.getMessage()==null)+":"+(absent.getMessage()==null)+":"+(value.getMessage()==text)+":"+(value!=other)+":"+calls+":"+value.getMessage().length()+":"+(int)value.getMessage().charAt(2));
 Throwable cause=new Exception("cause");System.out.println((empty.initCause(cause)==empty)+":"+(empty.getCause()==cause)+":"+(absent.initCause(cause)==absent));
 try{empty.initCause(null);System.out.println("bad");}catch(IllegalStateException expected){System.out.println("closed");}
 CancellationException generic=bound(text);Child child=new Child(text);System.out.println((generic.getMessage()==text)+":"+(child.getMessage()==text)+":"+(value.getClass()==CancellationException.class)+":"+(CancellationException.class.getSuperclass()==IllegalStateException.class));
 }
}`}, "probe.Main", "true:true:true:true:1:3:55296\ntrue:true:true\nclosed\ntrue:true:true:true\n")
}
func TestCampaignSecurityExceptionHierarchyJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>security-hierarchy</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
import java.security.GeneralSecurityException;import java.security.NoSuchAlgorithmException;
public class Main {
 public static void main(String[] args)throws Exception{
 NoSuchAlgorithmException value=new NoSuchAlgorithmException("detail");GeneralSecurityException base=value;
 System.out.println(value.getClass().getName()+":"+value.getClass().getSimpleName()+":"+(value.getClass()==NoSuchAlgorithmException.class));
 System.out.println((Class.forName("java.security.GeneralSecurityException")==GeneralSecurityException.class)+":"+(NoSuchAlgorithmException.class.getSuperclass()==GeneralSecurityException.class)+":"+(GeneralSecurityException.class.getSuperclass()==Exception.class));
 System.out.println(GeneralSecurityException.class.isAssignableFrom(NoSuchAlgorithmException.class)+":"+Exception.class.isAssignableFrom(NoSuchAlgorithmException.class)+":"+RuntimeException.class.isAssignableFrom(NoSuchAlgorithmException.class)+":"+NoSuchAlgorithmException.class.isAssignableFrom(GeneralSecurityException.class));
 try{throw value;}catch(GeneralSecurityException caught){System.out.println("security:"+(caught==value)+":"+(caught.getMessage()==value.getMessage()));}catch(Exception wrong){System.out.println("bad");}
 Object[] array=new GeneralSecurityException[2];array[0]=value;array[1]=new GeneralSecurityException();
 try{array[1]=new java.util.concurrent.CancellationException();System.out.println("bad-store");}catch(ArrayStoreException expected){System.out.println("store:"+(array[0]==base)+":"+(array[1].getClass()==GeneralSecurityException.class));}
 }
}`}, "probe.Main", "java.security.NoSuchAlgorithmException:NoSuchAlgorithmException:true\ntrue:true:true\ntrue:true:false:false\nsecurity:true:true\nstore:true:true\n")
}
