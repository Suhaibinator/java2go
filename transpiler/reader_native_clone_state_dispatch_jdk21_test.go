package transpiler

import "testing"

func TestReaderNativeCloneStateDispatchIdentityJDK21(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>reader</groupId><artifactId>clone-contract</artifactId><version>1</version></project>
`,
		"src/main/java/cloneapp/Derived.java": `package cloneapp;
import clonebase.Base;
public class Derived<T> extends Base<T> {
 public int own=11;
 public Derived(){super();}
 protected int code(){return own;}
}
`,
		"src/main/java/cloneapp/Main.java": `package cloneapp;
import clonebase.Base;
import java.io.Reader;
public final class Main {
 static class Aliased extends Reader implements java.lang.Cloneable {
  public Object ReaderBase=new Object();
  public int own=4;
  public int read(char[] b,int o,int n){if(n==0)return 0;b[o]=(char)own;return 1;}
  public void close(){}
  Object nativeLock(){return lock;}
  void nativeLock(Object value){lock=value;}
  Aliased copy()throws CloneNotSupportedException{return(Aliased)super.clone();}
 }
 static class AliasChild extends Aliased{public Object ReaderBase=new Object();}
 static class Denied extends Reader {
  public int read(char[]b,int o,int n){return -1;}public void close(){}
  Object copy()throws CloneNotSupportedException{return super.clone();}
 }
 public static void main(String[]args)throws Exception {
  Derived<String> original=new Derived<String>();
  int constructors=Base.constructed,initializers=Base.initialized;
  Base<String> copy=original.copy();
  Derived<String> derived=(Derived<String>)copy;
  derived.own=23;copy.state(19);
  System.out.println("hierarchy:"+(copy!=original)+":"+(copy.getClass()==original.getClass())+":"+copy.shallow(original)+":"+(constructors==Base.constructed)+":"+(initializers==Base.initialized));
  System.out.println("fields:"+original.state()+":"+copy.state()+":"+original.read()+":"+copy.read()+":"+original.sourceCalls+":"+copy.sourceCalls);
  System.out.println("default:"+(copy.nativeLock()==original));
  Object sentinel=new Object();original.nativeLock(sentinel);
  Base<String> other=original.copy();other.nativeLock(null);
  System.out.println("isolation:"+(copy.nativeLock()==original)+":"+(original.nativeLock()==sentinel)+":"+(other.nativeLock()==null));
  original.nativeLock(null);Base<String> nullable=original.copy();
  System.out.println("nullable:"+(nullable.nativeLock()==null)+":"+(constructors==Base.constructed));
  copy.userCallback.run();
  System.out.println("userClosure:"+original.callbackCalls()+":"+copy.callbackCalls()+":"+(original.userCallback==copy.userCallback));
  synchronized(original){System.out.println("monitorOriginal:"+Thread.holdsLock(original)+":"+Thread.holdsLock(copy));}
  synchronized(copy){System.out.println("monitorClone:"+Thread.holdsLock(original)+":"+Thread.holdsLock(copy));}
  Thread worker=new Thread(()->{try{Thread.currentThread().interrupt();int result=copy.read();System.out.println("callback:"+result+":"+copy.workerSeen+":"+copy.interruptedSeen+":"+(original.sourceCalls==1));}catch(Exception failure){throw new RuntimeException(failure);}finally{Thread.interrupted();}});
  worker.start();worker.join();
  AliasChild first=new AliasChild();Aliased alias=first.copy();alias.own=9;
  System.out.println("alias:"+(alias!=first)+":"+(alias.nativeLock()==first)+":"+(alias.ReaderBase==((Aliased)first).ReaderBase)+":"+first.read()+":"+alias.read());
  boolean denied=false;try{new Denied().copy();}catch(CloneNotSupportedException expected){denied=true;}
  System.out.println("denied:"+denied);
 }
}
`,
		"src/main/java/clonebase/Base.java": `package clonebase;
import java.io.Reader;
public class Base<T> extends Reader implements java.lang.Cloneable {
 public static int constructed;
 public static int initialized;
 private final Object shared = new Object();
 private final int[] values = new int[]{3};
 private final Base<T> initialSelf = this;
 private volatile int state = 7;
 private int callbackCalls;
 public int sourceCalls;
 public boolean workerSeen;
 public boolean interruptedSeen;
 private final Thread constructorThread = Thread.currentThread();
 public Runnable userCallback = () -> { callbackCalls++; };
 private final int field = ++initialized;
 public Base(){super();constructed++;}
 public Base(Object monitor){super(monitor);constructed++;}
 public int read(char[] buffer,int offset,int length) {
  sourceCalls++;
  workerSeen=Thread.currentThread()!=constructorThread;
  interruptedSeen=Thread.currentThread().isInterrupted();
  if(length==0)return 0;
  buffer[offset]=(char)code();return 1;
 }
 protected int code(){return state;}
 public void close(){}
 public Base<T> copy()throws CloneNotSupportedException{return(Base<T>)super.clone();}
 public Object nativeLock(){return lock;}
 public void nativeLock(Object value){lock=value;}
 public boolean shallow(Base<T> original){return shared==original.shared&&values==original.values&&initialSelf==original&&field==original.field;}
 public void state(int value){state=value;}
 public int state(){return state;}
 public int callbackCalls(){return callbackCalls;}
}
`,
	}
	expected := `hierarchy:true:true:true:true:true
fields:7:19:11:23:1:1
default:true
isolation:true:true:true
nullable:true:true
userClosure:1:0:true
monitorOriginal:true:false
monitorClone:false:true
callback:23:true:true:true
alias:true:true:true:4:9
denied:true
`
	runCampaignCompilerStrictProjectObservations(t, files, "cloneapp.Main", []campaignCompilerProjectObservation{{name: "first", stdout: expected}, {name: "second", stdout: expected}, {name: "third", stdout: expected}})
}
