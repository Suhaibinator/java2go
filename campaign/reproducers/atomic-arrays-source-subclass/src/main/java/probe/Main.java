package probe;import java.util.concurrent.atomic.*;public class Main {
 static class IntChild extends AtomicIntegerArray {IntChild(){super(2);}}
 static class LongChild extends AtomicLongArray {LongChild(){super(new long[]{23L});}}
 public static void main(String[] args){IntChild i=new IntChild();LongChild l=new LongChild();i.set(0,17);Object a=i;Object b=l;System.out.println(i.get(0)+":"+l.get(0)+":"+(a instanceof AtomicIntegerArray)+":"+(b instanceof java.io.Serializable));}
 }