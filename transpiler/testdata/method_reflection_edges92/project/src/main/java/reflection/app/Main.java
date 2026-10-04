package reflection.app;
import reflection.api.*;
import reflection.impl.*;
import java.lang.reflect.Method;
public class Main {
 public static void main(String[] args) throws Exception {
  Method inherited=Joined.class.getMethod("result");
  System.out.println("inherited:"+inherited.getReturnType().getName()+":"+inherited.getDeclaringClass().getName()+":"+inherited.isBridge()+":"+inherited.isSynthetic());
  Method hidden=PrivateBase.class.getDeclaredMethod("hidden");
  hidden.setAccessible(true);
  System.out.println("private:"+hidden.invoke(new PrivateChild()));
 }
}
