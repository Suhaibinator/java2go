import java.util.concurrent.atomic.AtomicIntegerFieldUpdater;
import java.util.concurrent.atomic.AtomicLongFieldUpdater;
import java.util.concurrent.atomic.AtomicReferenceFieldUpdater;
import java.util.function.IntUnaryOperator;
public class AtomicGeneratedWitness {
 public volatile int count;
 public volatile long stamp;
 public volatile String value;
 public int ordinary;
 public static volatile int shared;
 public final int fixed=7;
 static final AtomicIntegerFieldUpdater<AtomicGeneratedWitness> I=AtomicIntegerFieldUpdater.newUpdater(AtomicGeneratedWitness.class,"count");
 static final AtomicLongFieldUpdater<AtomicGeneratedWitness> L=AtomicLongFieldUpdater.newUpdater(AtomicGeneratedWitness.class,"stamp");
 static final AtomicReferenceFieldUpdater<AtomicGeneratedWitness,String> R=AtomicReferenceFieldUpdater.newUpdater(AtomicGeneratedWitness.class,String.class,"value");
 static class Increment implements IntUnaryOperator {public int applyAsInt(int x){return x+3;}}
 static void rejected(String field){try{AtomicIntegerFieldUpdater.newUpdater(AtomicGeneratedWitness.class,field);System.out.println("accepted");}catch(RuntimeException e){System.out.println(e.getClass().getSimpleName());}}
 public static void main(String[] args)throws Exception {
  AtomicGeneratedWitness x=new AtomicGeneratedWitness();
  System.out.println(I.get(x));I.set(x,3);System.out.println(x.count);x.count=7;System.out.println(I.compareAndSet(x,7,9));
  System.out.println(I.getAndAdd(x,2));System.out.println(I.incrementAndGet(x));
  I.set(x,Integer.MAX_VALUE);System.out.println(I.incrementAndGet(x));
  I.set(x,5);System.out.println(I.getAndUpdate(x,a->a+1));System.out.println(I.accumulateAndGet(x,4,(a,b)->a+b));
  System.out.println(I.updateAndGet(x,new Increment()));
  L.set(x,Long.MAX_VALUE);System.out.println(L.incrementAndGet(x));System.out.println(L.getAndAccumulate(x,3L,(a,b)->a+b));
  String a=new String("same"),b=new String("same");R.set(x,a);System.out.println(R.compareAndSet(x,b,b));System.out.println(R.get(x)==a);System.out.println(R.compareAndSet(x,a,b));System.out.println(x.value==b);
  System.out.println(R.updateAndGet(x,v->v)==b);R.lazySet(x,null);System.out.println(x.value==null);
  AtomicGeneratedWitness.class.getDeclaredField("count").set(x,Integer.valueOf(17));System.out.println(I.get(x));
  AtomicReferenceFieldUpdater raw=R;try{raw.set(x,Integer.valueOf(1));System.out.println("accepted");}catch(ClassCastException e){System.out.println("ClassCastException");}
  try{I.get(null);System.out.println("accepted");}catch(ClassCastException e){System.out.println("ClassCastException");}
  rejected("ordinary");rejected("shared");rejected("fixed");rejected("missing");
  try{AtomicIntegerFieldUpdater.newUpdater(AtomicGeneratedWitness.class,(String)null);System.out.println("accepted");}catch(RuntimeException e){System.out.println(e.getClass().getSimpleName());}
 }
}
