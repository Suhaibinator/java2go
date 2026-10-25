package records;
public class ReferenceProbe {
 static final Object LOCK=new Object();
 public record Value(int number){public Value{System.out.println("ref-held="+Thread.holdsLock(LOCK));number++;}}
 public record Empty(){}
 interface Maker {Value make(int n);}
 interface EmptyMaker {Empty make();}
 public static void main(String[] args){
  Maker compact=Value::new;EmptyMaker implicit=Empty::new;
  synchronized(LOCK){Value a=compact.make(4);Empty x=implicit.make();Empty y=implicit.make();System.out.println("ref="+a.number()+":"+(x!=y));}
 }
}
