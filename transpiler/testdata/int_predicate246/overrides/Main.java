package probe;import java.util.function.IntPredicate;
public class Main{
 static class Override implements IntPredicate{int calls;public boolean test(int n){return false;}public IntPredicate negate(){calls++;return n->false;}public IntPredicate and(IntPredicate other){calls+=10;return n->true;}public IntPredicate or(IntPredicate other){calls+=100;return n->false;}}
 interface Derived extends IntPredicate{default IntPredicate negate(){return n->false;}}
 static class FromDefault implements Derived{public boolean test(int n){return false;}}
 static class Covariant implements IntPredicate{static int calls;final boolean answer;Covariant(boolean value){answer=value;}public boolean test(int n){return answer;}public Covariant negate(){calls++;return new Covariant(!answer);}}
 public static void main(String[]args){Override source=new Override();IntPredicate p=source;System.out.println(p.negate().test(1)+":"+p.and(null).test(1)+":"+p.or(null).test(1)+":"+source.calls);IntPredicate inherited=new FromDefault();System.out.println(inherited.negate().test(1));IntPredicate covariant=new Covariant(false);System.out.println(covariant.negate().test(1)+":"+Covariant.calls);}
}
