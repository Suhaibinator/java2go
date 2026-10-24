package transpiler

import "testing"

func TestCampaignThrowableExplicitStringJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>throwablestring</groupId><artifactId>probe</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
class Message extends Exception {
 int calls; boolean held;
 Message(){super("stored");}
 public String getMessage(){calls++;held=Thread.holdsLock(this);return "virtual";}
}
class Localized extends Message {
 public String getLocalizedMessage(){calls++;held=Thread.holdsLock(this);return "localized";}
}
class Override extends RuntimeException {
 int calls;boolean held;String text="source";RuntimeException failure;
 public String toString(){calls++;held=Thread.holdsLock(this);if(failure!=null)throw failure;return text;}
 String parent(){return super.toString();}
}
public class Main {
 public static void main(String[] args){
  Exception checked=new Exception("checked");RuntimeException unchecked=new RuntimeException("unchecked");
  System.out.println(checked.toString()+"|"+unchecked.toString()+"|"+new Throwable().toString()+"|"+new Throwable("").toString());
  Exception absent=null;boolean npe=false;try{absent.toString();}catch(NullPointerException expected){npe=true;}System.out.println(npe);
  Message message=new Message();Throwable view=message;synchronized(message){System.out.println(view.toString()+":"+message.calls+":"+message.held);}
  Localized localized=new Localized();view=localized;synchronized(localized){System.out.println(view.toString()+":"+localized.calls+":"+localized.held);}
  Override source=new Override();view=source;synchronized(source){
   System.out.println(view.toString()+":"+source.calls+":"+source.held+":"+source.parent());
   source.text=null;System.out.println((view.toString()==null)+":"+source.calls);
   RuntimeException marker=new RuntimeException("marker");source.failure=marker;boolean same=false;try{view.toString();}catch(RuntimeException caught){same=caught==marker;}System.out.println(same+":"+source.calls);
  }
 }
}`,
	}, "probe.Main", "java.lang.Exception: checked|java.lang.RuntimeException: unchecked|java.lang.Throwable|java.lang.Throwable: \ntrue\nprobe.Message: virtual:1:true\nprobe.Localized: localized:1:true\nsource:1:true:probe.Override\ntrue:2\ntrue:3\n")
}

func TestCampaignThrowableExplicitStringSourceShadowJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                             `<project><modelVersion>4.0.0</modelVersion><groupId>throwablestring</groupId><artifactId>shadow</artifactId><version>1</version></project>`,
		"src/main/java/shadow/Exception.java": `package shadow; public class Exception {public String toString(){return "shadow";}}`,
		"src/main/java/probe/Main.java": `package probe;import shadow.Exception;
public class Main {public static void main(String[] args){Exception source=new Exception();Object object=source;java.lang.Exception builtin=new java.lang.Exception("builtin");System.out.println(source.toString()+":"+object.toString()+":"+builtin.toString());}}`,
	}, "probe.Main", "shadow:shadow:java.lang.Exception: builtin\n")
}

func TestCampaignThrowableInheritedDefaultStringJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>throwablestring</groupId><artifactId>inherited</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
class Plain extends Exception {Plain(){super("plain");}}
class Child extends Plain {}
class LocalizedDefault extends Exception {
 int calls;boolean held;
 public String getMessage(){calls++;held=Thread.holdsLock(this);return "message";}
 public String getLocalizedMessage(){return "local:"+super.getLocalizedMessage();}
}
public class Main {public static void main(String[] args){
 Throwable value=new Child();System.out.println(value.toString());
 LocalizedDefault source=new LocalizedDefault();value=source;synchronized(source){System.out.println(value.toString()+":"+source.calls+":"+source.held);}
}}
`,
	}, "probe.Main", "probe.Child: plain\nprobe.LocalizedDefault: local:message:1:true\n")
}
