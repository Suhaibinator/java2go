package probe;import java.util.function.IntPredicate;
public class Main{
 static class Check implements IntPredicate{final Object lock;Check(Object value){lock=value;}public boolean test(int n){return Thread.holdsLock(lock);}}
 static boolean free(int n){return true;}
 public static void main(String[]args){Object lock=new Object();IntPredicate source=new Check(lock);IntPredicate lambda=n->Thread.holdsLock(lock);IntPredicate composed=lambda.and(Main::free);synchronized(lock){System.out.println(source.test(1)+":"+source.negate().test(1)+":"+composed.test(1));}System.out.println(composed.test(1));}
}
