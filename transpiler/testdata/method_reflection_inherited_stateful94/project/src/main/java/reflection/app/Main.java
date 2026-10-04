package reflection.app;
import java.lang.reflect.Method;
import java.lang.reflect.InvocationTargetException;
import reflection.api.Base;
import reflection.impl.Child;
public class Main {
 public static void main(String[] args) throws Exception {
  Child child = new Child();
  Method narrow = Child.class.getDeclaredMethod("value");
  Method echo = Child.class.getDeclaredMethod("echo", String.class);
  System.out.println("value:" + narrow.getReturnType().getName() + ":" + narrow.isBridge() + ":" + narrow.isSynthetic() + ":" + narrow.getModifiers());
  System.out.println("echo:" + echo.getReturnType().getName() + ":" + echo.isBridge() + ":" + echo.isSynthetic() + ":" + echo.getModifiers());
  System.out.println("owners:" + (narrow.getDeclaringClass() == Child.class) + ":" + (echo.getDeclaringClass() == Child.class));
  Method setter = Base.class.getDeclaredMethod("set", Object.class);
  setter.invoke(child, "ok");
  System.out.println("value-result:" + (String) narrow.invoke(child));
  System.out.println("echo-result:" + (String) echo.invoke(child, "input"));
  try {
   echo.invoke(child, Integer.valueOf(7));
   throw new AssertionError("wrong reflection argument accepted");
  } catch (IllegalArgumentException expected) { System.out.println("argument-rejected"); }
  System.out.println("calls-before-pollution:" + child.calls);
  setter.invoke(child, Integer.valueOf(7));
  Method erased = Base.class.getDeclaredMethod("value");
  System.out.println("erased-result:" + erased.invoke(child));
  try {
   narrow.invoke(child);
   throw new AssertionError("bridge result cast absent");
  } catch (InvocationTargetException expected) {
   System.out.println("result-cause:" + expected.getCause().getClass().getName());
  }
  System.out.println("calls-after-pollution:" + child.calls);
 }
}
