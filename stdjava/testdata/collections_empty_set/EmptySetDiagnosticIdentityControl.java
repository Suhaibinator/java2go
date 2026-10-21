package emptysetnative;
import java.util.*;
public class EmptySetDiagnosticIdentityControl {
 static NullPointerException failure(Runnable action){try{action.run();}catch(NullPointerException failure){return failure;}throw new AssertionError("NPE required");}
 static void check(String name,Runnable action){
  NullPointerException first=failure(action),second=failure(action);
  String text=first.getMessage(),other=second.getMessage();
  System.out.println(name+"="+(text==first.getMessage())+":"+(text!=other)+":"+text.equals(other)+":"+(text!=text.intern())+":"+(text.intern()==other.intern()));
 }
 public static void main(String[] args){Set<Object> set=Collections.emptySet();check("contains",()->set.containsAll(null));check("array",()->set.toArray((Object[])null));}
}
