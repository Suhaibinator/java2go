package reflection.app;
import java.lang.reflect.Method;
import reflection.api.Base;
import reflection.impl.Child;
public class Main {
 public static void main(String[] args) throws Exception {
  Child child = new Child();
  Method method = Child.class.getDeclaredMethod("value");
  System.out.println("declared:" + method.getReturnType().getName() + ":" + method.isBridge() + ":" + method.isSynthetic() + ":" + method.getModifiers());
  System.out.println("owner:" + (method.getDeclaringClass() == Child.class));
  System.out.println("invoke-null:" + (method.invoke(child) == null));
  Method inherited = Base.class.getDeclaredMethod("value");
  System.out.println("base:" + inherited.getReturnType().getName() + ":" + inherited.isBridge() + ":" + inherited.isSynthetic());
  System.out.println("inherited-invoke-null:" + (inherited.invoke(child) == null));
 }
}
