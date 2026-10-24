import java.util.concurrent.*;
import java.util.concurrent.atomic.*;
import java.util.function.*;
public class OperatorExecutionCollisionProbe {
 public volatile int value;
 static final AtomicIntegerFieldUpdater<OperatorExecutionCollisionProbe> UPDATE=AtomicIntegerFieldUpdater.newUpdater(OperatorExecutionCollisionProbe.class,"value");
 static final IllegalArgumentException FAILURE=new IllegalArgumentException("callback");
 static String trace="";
 static class Collision implements IntUnaryOperator {
  public int applyAsIntJava2goExecution(int value){return -900;}
  public int applyAsIntJava2goExecution1(int value){return -901;}
  public int applyAsInt(int value){if(!Thread.holdsLock(this))throw new IllegalStateException("execution");if(value<0)throw FAILURE;return value+1;}
 }
 static AtomicIntegerFieldUpdater<OperatorExecutionCollisionProbe> receiver(){trace+="r";return UPDATE;}
 static OperatorExecutionCollisionProbe target(OperatorExecutionCollisionProbe value){trace+="t";return value;}
 static IntUnaryOperator callback(IntUnaryOperator value){trace+="f";return value;}
 public static void main(String[] args)throws Exception {
  Collision collision=new Collision();OperatorExecutionCollisionProbe target=new OperatorExecutionCollisionProbe();
  synchronized(collision){System.out.println(receiver().updateAndGet(target(target),callback(collision))+":"+trace);}
  UPDATE.set(target,-1);
  try{synchronized(collision){UPDATE.updateAndGet(target,collision);}}catch(IllegalArgumentException ex){System.out.println((ex==FAILURE)+":"+UPDATE.get(target));}
  synchronized(collision){UPDATE.set(target,4);System.out.println(UPDATE.updateAndGet(target,collision));}
  Object lock=new Object();IntUnaryOperator created=value->Thread.holdsLock(lock)?value+2:-99;
  ExecutorService pool=Executors.newSingleThreadExecutor();
  try{System.out.println(pool.submit(()->{synchronized(lock){return UPDATE.updateAndGet(target,created);}}).get(10,TimeUnit.SECONDS));}finally{pool.shutdownNow();}
 }
}
