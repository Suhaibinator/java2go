package records;
public class InitTokenProbe {
 static final Object LOCK=new Object();static boolean initialized;
 public record Value(){static{initialized=true;System.out.println("init-held="+Thread.holdsLock(LOCK));}}
 interface Maker {Value make();}
 public static void main(String[] args){
  Maker f=Value::new;System.out.println("before="+initialized);
  synchronized(LOCK){Value a=f.make();Value b=f.make();System.out.println("after="+initialized+":"+(a!=b));}
 }
}
