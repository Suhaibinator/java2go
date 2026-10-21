package transpiler

import "testing"

func TestCampaignNoSuchAlgorithmExceptionOverloadsJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>nsa-overloads</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
import java.security.NoSuchAlgorithmException;
import java.security.GeneralSecurityException;
class Child extends NoSuchAlgorithmException {
 <S extends String> Child(S message){super(message);}
 Child(Throwable cause){super(cause);}
 Child(String message,Throwable cause){super(message,cause);}
}
public class Main {
 static <S extends String> NoSuchAlgorithmException message(S value){return new NoSuchAlgorithmException(value);}
 static <T extends Throwable> NoSuchAlgorithmException cause(T value){return new NoSuchAlgorithmException(value);}
 static boolean available(NoSuchAlgorithmException value,Throwable replacement){
  try{return value.initCause(replacement)==value && value.getCause()==replacement;}
  catch(IllegalStateException expected){return false;}
 }
 public static void main(String[] args){
  String text=new String(new char[]{'r',0,(char)0xd800,'x',(char)0xdc00});
  RuntimeException replacement=new RuntimeException("replacement");
  NoSuchAlgorithmException empty=new NoSuchAlgorithmException();
  NoSuchAlgorithmException detail=new NoSuchAlgorithmException(text);
  NoSuchAlgorithmException absent=new NoSuchAlgorithmException((String)null);
  System.out.println("message="+(empty.getMessage()==null)+":"+(detail.getMessage()==text)+":"+(absent.getMessage()==null)+":"+available(empty,replacement)+":"+available(detail,replacement)+":"+available(absent,replacement));
  System.out.println("units="+detail.getMessage().length()+":"+(int)detail.getMessage().charAt(1)+":"+(int)detail.getMessage().charAt(2)+":"+(int)detail.getMessage().charAt(4));
  NoSuchAlgorithmException nullCause=new NoSuchAlgorithmException((Throwable)null);
  NoSuchAlgorithmException caused=new NoSuchAlgorithmException(replacement);
  NoSuchAlgorithmException paired=new NoSuchAlgorithmException(text,replacement);
  NoSuchAlgorithmException nullPair=new NoSuchAlgorithmException(null,null);
  NoSuchAlgorithmException nullMessage=new NoSuchAlgorithmException(null,replacement);
  System.out.println("cause="+(nullCause.getMessage()==null)+":"+(nullCause.getCause()==null)+":"+(caused.getCause()==replacement)+":"+(paired.getMessage()==text)+":"+(paired.getCause()==replacement)+":"+(nullPair.getMessage()==null)+":"+(nullMessage.getCause()==replacement));
  System.out.println("closed="+available(nullCause,replacement)+":"+available(caused,replacement)+":"+available(paired,replacement)+":"+available(nullPair,replacement)+":"+available(nullMessage,replacement));
  NoSuchAlgorithmException generic=message(text);
  NoSuchAlgorithmException genericNull=Main.<String>message(null);
  NoSuchAlgorithmException genericCause=cause(replacement);
  Child childMessage=new Child(text);Child childCause=new Child((Throwable)replacement);Child childPair=new Child(null,replacement);
  System.out.println("bound="+(generic.getMessage()==text)+":"+(genericNull.getMessage()==null)+":"+(genericCause.getCause()==replacement)+":"+(childMessage.getMessage()==text)+":"+(childCause.getCause()==replacement)+":"+(childPair.getCause()==replacement));
  System.out.println("class="+(detail.getClass()==NoSuchAlgorithmException.class)+":"+(NoSuchAlgorithmException.class.getSuperclass()==GeneralSecurityException.class));
 }
}`,
	}, "probe.Main", "message=true:true:true:true:true:true\nunits=5:0:55296:56320\ncause=true:true:true:true:true:true:true\nclosed=false:false:false:false:false\nbound=true:true:true:true:true:true\nclass=true:true\n")
}

func TestCampaignNoSuchAlgorithmExceptionCauseCallerJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>nsa-callback</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
import java.security.NoSuchAlgorithmException;
import java.security.GeneralSecurityException;
public class Main {
 static ThreadLocal<String> local=new ThreadLocal<String>();
 static String trace="";
 static String detail=new String(new char[]{'r',0,(char)0xd800});
 static RuntimeException marker=new RuntimeException("marker");
 static class Cause extends RuntimeException {
  int calls;boolean held;boolean same;boolean contextual;boolean abrupt;boolean nullText;Thread caller;
  public String toString(){calls++;held=Thread.holdsLock(this);same=Thread.currentThread()==caller;contextual=local.get()==detail;trace+="render,";if(abrupt)throw marker;return nullText?null:detail;}
 }
 static class Resource implements AutoCloseable {public void close(){trace+="close,";}}
 static String message(){trace+="message,";return detail;}
 static Throwable cause(Cause value){trace+="cause,";return value;}
 public static void main(String[] args){
  Cause value=new Cause();value.caller=Thread.currentThread();
  NoSuchAlgorithmException built=null;
  synchronized(value){local.set(detail);trace+="enter,";try(Resource resource=new Resource()){built=new NoSuchAlgorithmException(cause(value));trace+="built,";}finally{local.remove();trace+="finally,";}}
  synchronized(value){trace+="reacquire,";}
  System.out.println("caller="+(built.getMessage()==detail)+":"+(built.getCause()==value)+":"+value.calls+":"+value.held+":"+value.same+":"+value.contextual+":"+(local.get()==null)+":"+Thread.holdsLock(value));
  System.out.println(trace);
  trace="";Cause lazy=new Cause();lazy.abrupt=true;
  NoSuchAlgorithmException paired=new NoSuchAlgorithmException(message(),cause(lazy));
  System.out.println("pair="+(paired.getMessage()==detail)+":"+(paired.getCause()==lazy)+":"+lazy.calls+":"+trace);
  trace="";Cause abrupt=new Cause();abrupt.caller=Thread.currentThread();abrupt.abrupt=true;boolean caught=false;
  try{synchronized(abrupt){local.set(detail);trace+="enter,";try(Resource resource=new Resource()){new NoSuchAlgorithmException(cause(abrupt));trace+="bad,";}finally{local.remove();trace+="finally,";}}}
  catch(RuntimeException failure){caught=failure==marker;trace+="catch,";}
  synchronized(abrupt){trace+="reacquire,";}
  System.out.println("abrupt="+caught+":"+abrupt.calls+":"+abrupt.held+":"+abrupt.same+":"+abrupt.contextual+":"+(local.get()==null)+":"+Thread.holdsLock(abrupt));
  System.out.println(trace);
  Cause nil=new Cause();nil.nullText=true;NoSuchAlgorithmException nullText=new NoSuchAlgorithmException(nil);
  System.out.println("null-text="+(nullText.getMessage()==null)+":"+(nullText.getCause()==nil)+":"+nil.calls);
 }
}`,
	}, "probe.Main", "caller=true:true:1:true:true:true:true:false\nenter,cause,render,built,close,finally,reacquire,\npair=true:true:0:message,cause,\nabrupt=true:1:true:true:true:true:false\nenter,cause,render,close,finally,catch,reacquire,\nnull-text=true:true:1\n")
}
