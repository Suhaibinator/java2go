import java.util.function.*;
import java.util.concurrent.atomic.AtomicReferenceFieldUpdater;
public class NominalSamTransportProbe {
 static class Both implements IntUnaryOperator,IntBinaryOperator {
  public int applyAsInt(int a){return a+1;}
  public int applyAsInt(int a,int b){return a+b;}
 }
 static class Cell { volatile Object value; }
 static Object erased(Both value){return value;}
 public static void main(String[] args) {
  Both both=new Both();Object alias=both;
  IntUnaryOperator unary=(IntUnaryOperator)alias;
  IntBinaryOperator binary=(IntBinaryOperator)alias;
  System.out.println((alias==unary)+":"+(alias==binary)+":"+unary.applyAsInt(2)+":"+binary.applyAsInt(3,4));
  Object returned=erased(both);
  IntUnaryOperator returnedUnary=(IntUnaryOperator)returned;
  IntBinaryOperator returnedBinary=(IntBinaryOperator)returned;
  System.out.println((returned==returnedUnary)+":"+(returned==returnedBinary)+":"+returnedUnary.applyAsInt(8)+":"+returnedBinary.applyAsInt(6,7));
  Cell cell=new Cell();cell.value=both;
  AtomicReferenceFieldUpdater<Cell,Object> updater=AtomicReferenceFieldUpdater.newUpdater(Cell.class,Object.class,"value");
  Thread caller=Thread.currentThread();Thread[] seen=new Thread[1];
  Object old=updater.getAndUpdate(cell,value->{seen[0]=Thread.currentThread();return value;});
  IntUnaryOperator oldUnary=(IntUnaryOperator)old;
  IntBinaryOperator oldBinary=(IntBinaryOperator)old;
  System.out.println((old==both)+":"+(old==oldUnary)+":"+(old==oldBinary)+":"+oldUnary.applyAsInt(10)+":"+oldBinary.applyAsInt(8,9)+":"+(seen[0]==caller)+":"+(cell.value==both));
  Object absent=null;System.out.println((IntUnaryOperator)absent==null);
  try {IntUnaryOperator invalid=(IntUnaryOperator)new Object();System.out.println("accepted-wrong");}
  catch(ClassCastException expected){System.out.println("wrong");}
 }
}
