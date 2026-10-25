package dispatchproof;
public class AnonymousOverrideWitness {
 static int calls;
 static abstract class Gate {
  abstract int choose(Object value);
  int route(Object value){return choose(value);}
 }
 public static void main(String[] args){
  final int offset=40;
  Gate gate=new Gate(){public int choose(Object value){calls++;return offset+(value==null?1:2);}};
  System.out.println(gate.choose(null)+":"+gate.route(new Object())+":"+calls);
 }
}
