import java.util.concurrent.atomic.AtomicIntegerFieldUpdater;
public class AtomicUpdaterColdFactoryProbe {
 static int initialized;
 public static class Cold {static {initialized=initialized+1;}public volatile int count;}
 static AtomicIntegerFieldUpdater<Cold> factory(){return AtomicIntegerFieldUpdater.newUpdater(Cold.class,"count");}
 static int factoryJava2goExecution(){return -93;}
 public static void main(String[] args){
  Class<Cold> literal=Cold.class;System.out.println("literal:"+initialized);
  AtomicIntegerFieldUpdater<Cold> updater=factory();System.out.println("factory:"+initialized);
  Cold target=new Cold();System.out.println("new:"+initialized);
  updater.set(target,4);System.out.println("value:"+target.count);
  AtomicIntegerFieldUpdater<Cold> second=AtomicIntegerFieldUpdater.newUpdater(literal,"count");
  second.set(target,7);System.out.println("same:"+updater.get(target)+":"+initialized);
 }
}
