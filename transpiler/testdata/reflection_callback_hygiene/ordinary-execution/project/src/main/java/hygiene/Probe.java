package hygiene;
import java.lang.reflect.Field;
public class Probe {
 static Thread execution;
 static Thread execution0,execution1,value0,value1,converted0,converted1,__java2goExecution,__java2goExecution0,__java2goExecution1;
 static final Thread stable=Thread.currentThread();
 public static void main(String[] args)throws Exception {
  Thread current=Thread.currentThread();
  Field selected=Probe.class.getDeclaredField("execution");
  System.out.println("before="+(selected.get(null)==null));
  selected.set(null,current);
  System.out.println("backing="+(execution==current));
  System.out.println("read="+(selected.get(null)==current));
  selected.set(null,null);
  System.out.println("cleared="+(execution==null));
  System.out.println("suffix="+(execution0==null && execution1==null && value0==null && value1==null && converted0==null && converted1==null && __java2goExecution==null && __java2goExecution0==null && __java2goExecution1==null));
  boolean denied=false;
  try{Probe.class.getDeclaredField("stable").set(null,current);}catch(IllegalAccessException expected){denied=true;}
  System.out.println("final="+denied+":"+(stable==current));
 }
}
