package transpiler

import "testing"

func TestCampaignIllegalArgumentConstructorReferenceJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>illegal-argument</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
class Cause extends RuntimeException {
 int calls;boolean held;String text="virtual";RuntimeException abrupt;
 public String toString(){calls++;held=Thread.holdsLock(this);if(abrupt!=null)throw abrupt;return text;}
}
public class Main {
 public static void main(String[] args){
  String message=new String(new char[]{'b','a','d','=','7','\ud800'});
  Throwable cause=new Throwable("root");
  IllegalArgumentException detail=new IllegalArgumentException(message),empty=new IllegalArgumentException(),nullMessage=new IllegalArgumentException((String)null);
  System.out.println((detail.getMessage()==message)+":"+(empty.getMessage()==null)+":"+(nullMessage.getMessage()==null)+":"+(detail.initCause(cause)==detail)+":"+(empty.initCause(cause)==empty)+":"+(nullMessage.initCause(cause)==nullMessage));
  IllegalArgumentException nullCause=new IllegalArgumentException((Throwable)null),pair=new IllegalArgumentException(message,cause),nullPair=new IllegalArgumentException((String)null,(Throwable)null);
  boolean blocked=false,pairBlocked=false;try{nullCause.initCause(cause);}catch(IllegalStateException expected){blocked=true;}try{nullPair.initCause(cause);}catch(IllegalStateException expected){pairBlocked=true;}
  System.out.println((nullCause.getMessage()==null)+":"+blocked+":"+(pair.getMessage()==message)+":"+(pair.getCause()==cause)+":"+(nullPair.getMessage()==null)+":"+pairBlocked);
  Cause source=new Cause();synchronized(source){
   IllegalArgumentException from=new IllegalArgumentException((Throwable)source);
   System.out.println(from.getMessage()+":"+(from.getCause()==source)+":"+source.calls+":"+source.held);
   source.text=null;IllegalArgumentException absent=new IllegalArgumentException((Throwable)source);
   RuntimeException marker=new RuntimeException("marker");source.abrupt=marker;boolean same=false;try{new IllegalArgumentException((Throwable)source);}catch(RuntimeException thrown){same=thrown==marker;}
   System.out.println((absent.getMessage()==null)+":"+(absent.getCause()==source)+":"+same+":"+source.calls+":"+source.held);
  }
 }
}`,
	}, "probe.Main", "true:true:true:true:true:true\ntrue:true:true:true:true:true\nvirtual:true:1:true\ntrue:true:true:3:true\n")
}
