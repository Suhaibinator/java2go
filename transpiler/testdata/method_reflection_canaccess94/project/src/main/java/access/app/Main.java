package access.app;
import access.api.Owner;
import java.lang.reflect.Method;
public class Main {
 static void check(String label,Method method,Object receiver){
  try {System.out.println(label+":"+method.canAccess(receiver)+":"+Owner.calls);}
  catch(IllegalArgumentException expected){System.out.println(label+":illegal-argument:"+Owner.calls);}
 }
 public static void main(String[] args) throws Exception {
  Owner owner=new Owner();
  Method staticMethod=Owner.class.getMethod("staticValue");
  Method instanceMethod=Owner.class.getMethod("instanceValue");
  check("static-null",staticMethod,null);
  check("static-nonnull",staticMethod,owner);
  check("instance-valid",instanceMethod,owner);
  check("instance-null",instanceMethod,null);
  check("instance-wrong",instanceMethod,new Object());
  staticMethod.setAccessible(true);instanceMethod.setAccessible(true);
  check("static-override-nonnull",staticMethod,owner);
  check("instance-override-null",instanceMethod,null);
  System.out.println("invoke:"+staticMethod.invoke(owner)+":"+instanceMethod.invoke(owner)+":"+Owner.calls);
 }
}
