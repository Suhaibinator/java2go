package parity.atomic_text_results;
import java.util.concurrent.atomic.*;
public class AtomicTextResults {
 public static void main(String[] args) {
  AtomicInteger i=new AtomicInteger(3);
  System.out.println("i:"+i.get()+":"+i.incrementAndGet()+":"+i.decrementAndGet()+":"+i.getAndIncrement()+":"+i.getAndDecrement()+":"+i.addAndGet(4)+":"+i.getAndAdd(2)+":"+i.compareAndSet(9,11)+":"+i.get());
  AtomicLong l=new AtomicLong(2147483648L);
  System.out.println("l:"+l.get()+":"+l.incrementAndGet()+":"+l.decrementAndGet()+":"+l.getAndIncrement()+":"+l.getAndDecrement()+":"+l.addAndGet(4L)+":"+l.getAndAdd(2L)+":"+l.compareAndSet(2147483654L,19L)+":"+l.get());
  AtomicBoolean b=new AtomicBoolean();
  System.out.println("b:"+b.get()+":"+b.compareAndSet(false,true)+":"+b.get());
  Object boxed=i.get(); System.out.println("boxed:"+boxed+":"+String.valueOf(l.get()));
 }
}
