package probe;
import java.util.concurrent.CountDownLatch;
public class Main {
 static java.lang.Thread Thread;
 public static void main(String[] args) throws Exception {
  System.out.println("foreign:"+new foreign.Thread().interrupt());
  System.out.println("source:"+new shadow.Thread().interrupt());
  System.out.println("binder:"+new Binder<foreign.Thread>().run(new foreign.Thread()));
  Thread=java.lang.Thread.currentThread();Thread.interrupt();
  try{new CountDownLatch(1).await();System.out.println("field:missed");}catch(InterruptedException expected){System.out.println("field:consumed");}
 }
}
