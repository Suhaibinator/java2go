package probe;import java.util.function.IntPredicate;
public class Main{
 interface Derived extends IntPredicate{}
 static abstract class Base implements IntPredicate{}
 static class Child extends Base implements Derived{public boolean test(int n){return n>0;}}
 public static void main(String[]args){Derived d=new Child();Base b=new Child();System.out.println(d.negate().test(1)+":"+b.negate().test(-1)+":"+d.and(n->n<5).test(2)+":"+b.or(n->n<0).test(-2));}
}
