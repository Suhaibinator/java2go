package probe;import java.util.function.IntPredicate;
public class Main{
 static class Check implements IntPredicate{int seen;public boolean test(int n){seen++;return n>0;}}
 static class Override implements IntPredicate{public boolean test(int n){return false;}public IntPredicate negate(){return n->true;}}
 public static void main(String[]args){Check source=new Check();IntPredicate p=source.and(n->n<3);System.out.println(p.test(2)+":"+p.test(4)+":"+source.seen);Override override=new Override();System.out.println(override.negate().test(1));}
}
