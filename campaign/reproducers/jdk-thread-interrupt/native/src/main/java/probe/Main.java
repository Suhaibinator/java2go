package probe;
import java.util.concurrent.CountDownLatch;
public class Main {
 static int evaluations;
 static Thread target(Thread t){evaluations++;return t;}
 static void consume(String label) {
  try{new CountDownLatch(1).await();System.out.println(label+":missed");}
  catch(InterruptedException expected){System.out.println(label+":consumed");}
 }
 public static void main(String[] args) throws Exception {
  Thread self=Thread.currentThread();target(self).interrupt();self.interrupt();consume("current");
  self.interrupt();consume("reset");System.out.println("once:"+evaluations);
  Thread fresh=new Thread(()->consume("new"));fresh.interrupt();fresh.interrupt();fresh.start();fresh.join();
  CountDownLatch ready=new CountDownLatch(1);
  Thread alive=new Thread(()->{ready.countDown();consume("alive");});alive.start();ready.await();alive.interrupt();alive.join();
  Thread ended=new Thread();ended.start();ended.join();ended.interrupt();ended.interrupt();System.out.println("terminated:"+ended.isAlive());
  try{target(null).interrupt();System.out.println("null:missed");}catch(NullPointerException expected){System.out.println("null:"+expected.getClass().getName());}
  System.out.println("evaluations:"+evaluations);
 }
}
