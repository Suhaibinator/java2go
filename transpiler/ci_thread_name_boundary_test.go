package transpiler

import "testing"

func TestCIThreadCanonicalNameIdentityAndExecution(t *testing.T) {
	const source = `import java.util.concurrent.*;
public class CIThreadCanonicalName {
 public static String run() throws Exception {
  java.lang.Thread caller=java.lang.Thread.currentThread();
  String main=caller.getName();
  boolean[] observed=new boolean[3];
  Thread worker=new Thread(() -> {
   Thread current=Thread.currentThread();
   String name=current.getName();
   observed[0]=name==current.getName();
   observed[1]=current!=caller;
   observed[2]=name!=main;
  });
  String before=worker.getName();worker.start();worker.join();
  ExecutorService pool=Executors.newSingleThreadExecutor();
  Future<String> one=pool.submit(() -> Thread.currentThread().getName());
  Future<String> two=pool.submit(() -> Thread.currentThread().getName());
  String poolName=one.get();boolean samePool=poolName==two.get();
  pool.shutdown();pool.awaitTermination(10,TimeUnit.SECONDS);
  boolean nullReceiver=false;try {Thread absent=null;absent.getName();}catch(NullPointerException ex){nullReceiver=true;}
  return main.equals("main")+":"+(main=="main")+":"+(main==caller.getName())+":"
   +(before==worker.getName())+":"+observed[0]+":"+observed[1]+":"+observed[2]+":"
   +samePool+":"+(poolName!=main)+":"+nullReceiver;
 }
}`
	verifyCanonicalStringStreamOracle(t, "CIThreadCanonicalName", source)
}

func TestCIThreadCanonicalOwnerPreservesSourceShadow(t *testing.T) {
	const source = `public class CIThreadNameOwner {
 static class Thread {String getName(){return "source";}}
 public static String run(){return new Thread().getName()+":"+java.lang.Thread.currentThread().getName().equals("main");}
}`
	verifyCanonicalStringStreamOracle(t, "CIThreadNameOwner", source)
}

func TestCIThreadNamedConstructorCanonicalUTF16(t *testing.T) {
	const source = `public class CIThreadNamedUTF16 {
 public static String run() throws Exception {
  String supplied=new String(new char[]{'A',0,'\ud800','\udc00','\udc00'});
  Thread named=new Thread(supplied);
  boolean[] observed=new boolean[2];
  Thread task=new Thread(() -> {
   String name=Thread.currentThread().getName();
   observed[0]=name==supplied;observed[1]=name.charAt(4)=='\udc00';
  },supplied);
  task.start();task.join();
  boolean oneNull=false;boolean twoNull=false;
  String missing=null;
  try{new Thread(missing);}catch(NullPointerException ex){oneNull=true;}
  try{new Thread(() -> {},missing);}catch(NullPointerException ex){twoNull=true;}
  return (named.getName()==supplied)+":"+(task.getName()==supplied)+":"+supplied.length()+":"
   +(int)named.getName().charAt(1)+":"+(int)named.getName().charAt(4)+":"
   +observed[0]+":"+observed[1]+":"+oneNull+":"+twoNull;
 }
}`
	verifyCanonicalStringStreamOracle(t, "CIThreadNamedUTF16", source)
}

func TestCIThreadNamedSourceRunnableExecution(t *testing.T) {
	const source = `public class CIThreadNamedSourceRunnable {
 static class Work implements Runnable {
  Thread caller;String seen;boolean own;int calls;
  Work(Thread main){caller=main;}
  public void run(){calls++;Thread current=Thread.currentThread();seen=current.getName();own=current!=caller;}
 }
 public static String run() throws Exception {
  String name=new String(new char[]{'R','\ud800'});
  Work work=new Work(Thread.currentThread());Thread task=new Thread(work,name);
  task.start();task.join();return (work.seen==name)+":"+work.own+":"+work.calls+":"+(int)work.seen.charAt(1);
 }
}`
	verifyCanonicalStringStreamOracle(t, "CIThreadNamedSourceRunnable", source)
}
