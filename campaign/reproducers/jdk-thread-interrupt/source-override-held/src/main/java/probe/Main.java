package probe;
public class Main {
 static final ThreadLocal<Integer> local=new ThreadLocal<>();
 static final RuntimeException sentinel=new RuntimeException("abrupt");
 static class Child extends Thread {
  int calls;boolean withSuper,throwing;Thread caller;
  public synchronized void interrupt(){calls++;if(Thread.currentThread()!=caller||local.get()!=41)throw new AssertionError("caller");if(throwing)throw sentinel;if(withSuper)super.interrupt();}
 }
 public static void main(String[] args){
  local.set(41);Child child=new Child();child.caller=Thread.currentThread();Thread base=child;
  base.interrupt();base.interrupt();System.out.println("no-super:"+child.calls+":"+child.isInterrupted());
  child.withSuper=true;base.interrupt();base.interrupt();System.out.println("with-super:"+child.calls+":"+child.isInterrupted());
  child.throwing=true;try{base.interrupt();}catch(RuntimeException e){System.out.println("pending-abrupt:"+(e==sentinel)+":"+child.calls+":"+child.isInterrupted());}
  Child clean=new Child();clean.caller=Thread.currentThread();clean.throwing=true;try{clean.interrupt();}catch(RuntimeException e){System.out.println("clean-abrupt:"+(e==sentinel)+":"+clean.calls+":"+clean.isInterrupted());}
 }
}
