package probe;import java.util.function.IntPredicate;
public class Main{
 static int calls;
 static boolean odd(int n){calls++;return n%2!=0;}
 static class Check{final int lower;Check(int n){lower=n;}boolean accepts(int n){calls++;return n>=lower;}}
 public static void main(String[]args){IntPredicate negative=n->n<0;IntPredicate odd=Main::odd;Check receiver=new Check(3);IntPredicate bound=receiver::accepts;System.out.println(negative.test(-1)+":"+odd.test(3)+":"+odd.test(4)+":"+bound.test(3)+":"+calls);}
}
