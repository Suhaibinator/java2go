package probe;import java.util.function.IntPredicate;import java.util.function.IntUnaryOperator;
public class Main{
 static class Check implements IntPredicate,IntUnaryOperator{int seen;public boolean test(int n){seen+=n;return n%2==0;}public int applyAsInt(int n){seen+=n;return seen;}}
 interface Derived extends IntPredicate{}
 static class Child implements Derived{public boolean test(int n){return n>0;}}
 public static void main(String[]args){Check source=new Check();IntPredicate p=source;Object erased=source;IntPredicate q=(IntPredicate)erased;IntUnaryOperator u=source;System.out.println((p==q)+":"+p.test(2)+":"+u.applyAsInt(6));System.out.println(q.negate().test(2)+":"+source.seen);Derived child=new Child();IntPredicate inherited=child;System.out.println(inherited.test(1)+":"+u.applyAsInt(2));}
}
