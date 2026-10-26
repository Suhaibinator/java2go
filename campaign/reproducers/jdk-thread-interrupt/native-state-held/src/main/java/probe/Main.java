package probe;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.atomic.AtomicBoolean;
public class Main {
 public static void main(String[] args) throws Exception {
  Thread fresh=new Thread();
  System.out.println("new-initial:"+fresh.isInterrupted()+":"+fresh.isAlive());
  fresh.interrupt();fresh.interrupt();
  System.out.println("new-repeated:"+fresh.isInterrupted()+":"+fresh.isAlive());
  fresh.start();fresh.join();
  System.out.println("terminated-retained:"+fresh.isInterrupted()+":"+fresh.isAlive());
  Thread ended=new Thread();ended.start();ended.join();ended.interrupt();ended.interrupt();
  System.out.println("terminated-request:"+ended.isInterrupted()+":"+ended.isAlive());
  CountDownLatch ready=new CountDownLatch(1);AtomicBoolean release=new AtomicBoolean(false);
  Thread worker=new Thread(()->{ready.countDown();while(!release.get())Thread.onSpinWait();
   System.out.println("alive:"+Thread.currentThread().isInterrupted()+":"+Thread.interrupted()+":"+Thread.interrupted()+":"+Thread.currentThread().isInterrupted());});
  worker.start();ready.await();worker.interrupt();worker.interrupt();release.set(true);worker.join();
  Thread current=Thread.currentThread();current.interrupt();current.interrupt();
  System.out.println("current:"+current.isInterrupted()+":"+Thread.interrupted()+":"+Thread.interrupted()+":"+current.isInterrupted());
  current.interrupt();System.out.println("reset:"+Thread.interrupted()+":"+Thread.interrupted());
  try{((Thread)null).interrupt();}catch(NullPointerException expected){System.out.println("null:"+expected.getClass().getName());}
 }
}
