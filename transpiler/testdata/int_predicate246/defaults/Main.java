package probe;import java.util.function.IntPredicate;
public class Main{
 static int left,right,effects;
 static IntPredicate source(boolean value){return n->{left++;return value;};}
 static IntPredicate other(boolean value){return n->{right++;return value;};}
 static IntPredicate absent(){effects++;return null;}
 static int argument(){effects++;return 3;}
 public static void main(String[]args){
 IntPredicate f=source(false),t=source(true);IntPredicate r=other(true);
 System.out.println(f.and(r).test(1)+":"+left+":"+right);left=0;right=0;
 System.out.println(t.or(r).test(1)+":"+left+":"+right);left=0;right=0;
 System.out.println(t.and(other(false)).test(1)+":"+left+":"+right);left=0;right=0;
 System.out.println(f.negate().test(1)+":"+left+":"+right);
 try{f.and(null);System.out.println("missing-right");}catch(NullPointerException expected){System.out.println("right-null");}
 IntPredicate nil=null;try{nil.or(absent());System.out.println("missing-left");}catch(NullPointerException expected){System.out.println("left-null:"+effects);}
 effects=0;try{nil.test(argument());System.out.println("missing-test");}catch(NullPointerException expected){System.out.println("argument:"+effects);}
 }
}
