import java.util.concurrent.atomic.AtomicIntegerFieldUpdater;
public class AtomicGeneratedOrderingWitness {
 static int initialized;
 static int trace;
 public static class Cold {static {initialized=initialized+1;}public volatile int count;}
 static AtomicIntegerFieldUpdater<Cold> receiver(){trace=trace*10+1;return null;}
 static Cold target(){trace=trace*10+2;return null;}
 static int argument(){trace=trace*10+3;return 8;}

 public static void main(String[] args){
  Class<Cold> type=Cold.class;System.out.println(initialized);
  AtomicIntegerFieldUpdater<Cold> updater=AtomicIntegerFieldUpdater.newUpdater(type,"count");System.out.println(initialized);
  Cold x=new Cold();System.out.println(initialized);updater.set(x,4);System.out.println(x.count);
  try{receiver().set(target(),argument());System.out.println("accepted");}catch(NullPointerException e){System.out.println(trace);}
  try{updater.getAndUpdate(null,null);System.out.println("accepted");}catch(ClassCastException e){System.out.println("target-first");}
  try{updater.getAndUpdate(x,null);System.out.println("accepted");}catch(NullPointerException e){System.out.println("callback-null");}
  try{AtomicIntegerFieldUpdater.newUpdater(PrivateOwner.class,"hidden");System.out.println("accepted");}catch(RuntimeException e){System.out.println("RuntimeException");}
 }
}

class PrivateOwner {private volatile int hidden;}
