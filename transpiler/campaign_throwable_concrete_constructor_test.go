package transpiler

import "testing"

func TestCampaignConcreteThrowablePublicConstructorsJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>throwableprobe</groupId><artifactId>constructors</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
public class Main {
 public static void main(String[] args){
  Throwable cause=new Throwable("cause");
  Throwable empty=new Throwable();Throwable message=new Throwable("detail");
  Throwable from=new Throwable(cause);Throwable pair=new Throwable("pair",cause);
  Throwable nullMessage=new Throwable((String)null);Throwable nullCause=new Throwable((Throwable)null);
  System.out.println((empty.getMessage()==null)+":"+(empty.getCause()==null)+":"+message.getMessage()+":"+from.getMessage()+":"+pair.getMessage()+":"+(from.getCause()==cause)+":"+(pair.getCause()==cause));
  boolean emptyInit=empty.initCause(cause)==empty;boolean messageInit=nullMessage.initCause(cause)==nullMessage;
  boolean blocked=false;try{nullCause.initCause(cause);}catch(IllegalStateException expected){blocked=true;}
  Throwable[] values={empty,message,from,pair,nullMessage,nullCause};Object[] objects=values;
  System.out.println(emptyInit+":"+messageInit+":"+blocked+":"+(objects[2]==from)+":"+from.getClass().getName());
 }
}`,
	}, "probe.Main", "true:true:detail:java.lang.Throwable: cause:pair:true:true\ntrue:true:true:true:java.lang.Throwable\n")
}

func TestCampaignConcreteThrowableSourceSubclassJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>throwableprobe</groupId><artifactId>subclasses</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
class DefaultCause extends Throwable {}
class BaseCause extends Throwable {
 BaseCause(String message){super(message);}
 BaseCause(Throwable cause){super(cause);}
 BaseCause(String message,Throwable cause){super(message,cause);}
}
class SourceCause extends Throwable {
 int calls;boolean held;String text="source";RuntimeException abrupt;
 SourceCause(){super("stored");}
 public String getMessage(){return "override:"+super.getMessage();}
 public String toString(){calls++;held=Thread.holdsLock(this);if(abrupt!=null)throw abrupt;return text;}
}
class InheritedCause extends SourceCause {}
public class Main {
 public static void main(String[] args){
  Throwable cause=new Throwable("root");BaseCause base=new BaseCause("base"),pair=new BaseCause("pair",cause),from=new BaseCause(cause);DefaultCause empty=new DefaultCause();
  System.out.println((empty.getMessage()==null)+":"+base.getMessage()+":"+pair.getMessage()+":"+(pair.getCause()==cause)+":"+from.getMessage()+":"+(from.getCause()==cause));
  SourceCause source=new InheritedCause();Throwable view=source;
  synchronized(source){
   Throwable wrapped=new Throwable(view);System.out.println(view.getMessage()+":"+wrapped.getMessage()+":"+(wrapped.getCause()==source)+":"+source.calls+":"+source.held);
   source.text=null;Throwable absent=new Throwable(view);System.out.println((absent.getMessage()==null)+":"+(absent.getCause()==source)+":"+source.calls+":"+source.held);
   RuntimeException marker=new RuntimeException("marker");source.abrupt=marker;boolean same=false;try{new Throwable(view);}catch(RuntimeException thrown){same=thrown==marker;}
   System.out.println(same+":"+source.calls+":"+source.held);
  }
 }
}`,
	}, "probe.Main", "true:base:pair:true:java.lang.Throwable: root:true\noverride:stored:source:true:1:true\ntrue:true:2:true\ntrue:3:true\n")
}
