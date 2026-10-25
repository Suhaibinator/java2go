package records;
public class ExecutionTokenProbe {
 static final Object LOCK=new Object();
 public record Value(int number) {
  public Value {System.out.println("held="+Thread.holdsLock(LOCK));}
  public Value(){this(3);System.out.println("suffix="+Thread.holdsLock(LOCK));}
 }
 public static void main(String[] args){synchronized(LOCK){new Value(1);new Value();}}
}
