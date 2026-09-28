package transpiler

import (
	"fmt"
	"testing"
)

func campaignMonitorOracle(t *testing.T, className, source string) {
	t.Helper()
	want := campaignRuntimeJavaOracle(t, className, source)
	t.Logf("JVM monitor oracle: %s", want)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("testing";"time")
func TestMonitor(t *testing.T){done:=make(chan string,1);go func(){done<-Run()}();select{case got:=<-done:if got!=%q{t.Fatalf("JVM %%q != Go %%q",%q,got)};case <-time.After(5*time.Second):t.Fatal("monitor execution did not complete")}}`, want, want))
}

func TestCampaignRuntimeMonitorEdges(t *testing.T) {
	const source = `class ObservationBase {
    public boolean checkAlias(Object other) {
        synchronized (this) { return Thread.holdsLock(other); }
    }
}
class ObservationDerived extends ObservationBase {}
public final class MonitorEdges {
    public static String run() {
        ObservationDerived derived = new ObservationDerived();
        ObservationBase base = derived;
        boolean baseHeld;
        boolean derivedHeld;
        synchronized (base) {
            baseHeld = Thread.holdsLock(base);
            derivedHeld = Thread.holdsLock(derived);
        }
        String absent = null;
        String nullResult;
        try { nullResult = "returned-" + Thread.holdsLock(absent); }
        catch (NullPointerException expected) { nullResult = "NPE"; }
        return base.checkAlias(derived) + ":" + baseHeld + ":" + derivedHeld + ":" + nullResult;
    }
}`
	campaignMonitorOracle(t, "MonitorEdges", source)
}

func TestCampaignRuntimeMonitorInheritedMethod(t *testing.T) {
	const source = `class MethodMonitorBase {
 public synchronized boolean owns(Object alias) {return Thread.holdsLock(alias);}
 public synchronized boolean reenter(MethodMonitorBase alias){return alias.owns(this);}
}
class MethodMonitorChild extends MethodMonitorBase {}
public class CampaignMonitorInherited {
 public static String run(){MethodMonitorChild child=new MethodMonitorChild();MethodMonitorBase base=child;return base.owns(child)+":"+child.reenter(base)+":"+Thread.holdsLock(child);}
}`
	campaignMonitorOracle(t, "CampaignMonitorInherited", source)
}

func TestCampaignRuntimeMonitorWaitAliases(t *testing.T) {
	const source = `import java.util.concurrent.CountDownLatch;
class WaitMonitorBase {
 boolean proceed;String result="";
 public synchronized void awaitAlias(Object alias,CountDownLatch ready){
  synchronized(this){ready.countDown();try{while(!proceed){alias.wait();}}catch(InterruptedException failure){throw new RuntimeException(failure);}
   result=Thread.holdsLock(alias)+":"+Thread.holdsLock(this);
  }
  result+=":"+Thread.holdsLock(alias);
 }
}
class WaitMonitorChild extends WaitMonitorBase {}
public class CampaignMonitorWait {
 static String one(boolean all)throws Exception{
  WaitMonitorChild child=new WaitMonitorChild();WaitMonitorBase base=child;CountDownLatch ready=new CountDownLatch(1);
  Thread worker=new Thread(()->base.awaitAlias(child,ready));worker.start();ready.await();
  synchronized(child){base.proceed=true;if(all){base.notifyAll();}else{base.notify();}}
  worker.join();return base.result+":"+Thread.holdsLock(child);
 }
 public static String run()throws Exception{return one(false)+"/"+one(true);}
}`
	campaignMonitorOracle(t, "CampaignMonitorWait", source)
}

func TestCampaignRuntimeMonitorConstructorCallback(t *testing.T) {
	const source = `class ConstructMonitorBase {
 boolean held;
 ConstructMonitorBase(){synchronized(this){held=check();}}
 public boolean check(){return false;}
}
class ConstructMonitorChild extends ConstructMonitorBase {
 public boolean check(){return Thread.holdsLock(this);}
}
public class CampaignMonitorConstructor {
 public static String run(){ConstructMonitorChild child=new ConstructMonitorChild();return child.held+":"+Thread.holdsLock(child);}
}`
	campaignMonitorOracle(t, "CampaignMonitorConstructor", source)
}

func TestCampaignRuntimeMonitorNullReferences(t *testing.T) {
	const source = `public class CampaignMonitorNull {
 public static String run()throws Exception {
  String absent=null;Object plain=null;int failures=0;
  try{synchronized(absent){failures=-100;}}catch(NullPointerException expected){failures++;}
  try{absent.wait();}catch(NullPointerException expected){failures++;}
  try{absent.notify();}catch(NullPointerException expected){failures++;}
  try{absent.notifyAll();}catch(NullPointerException expected){failures++;}
  try{Thread.holdsLock(absent);}catch(NullPointerException expected){failures++;}
  try{synchronized(plain){failures=-100;}}catch(NullPointerException expected){failures++;}
  try{plain.wait();}catch(NullPointerException expected){failures++;}
  try{plain.notify();}catch(NullPointerException expected){failures++;}
  try{plain.notifyAll();}catch(NullPointerException expected){failures++;}
  try{Thread.holdsLock(plain);}catch(NullPointerException expected){failures++;}
  return "NPE:"+failures;
 }
}`
	campaignMonitorOracle(t, "CampaignMonitorNull", source)
}
