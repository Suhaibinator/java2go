package reflection.app;
import java.lang.reflect.Method;
import reflection.api.*;
public class Main {
 public static void main(String[] args) throws Exception {
  System.out.println("class-real:"+Leaf.class.getMethod("flag").getModifiers());
  for (Method method:Leaf.class.getDeclaredMethods()) {
   if (method.isBridge()) System.out.println("class-bridge:"+method.getModifiers()+":"+method.getReturnType().getName());
  }
  System.out.println("default-real:"+PlainDefault.class.getMethod("flag").getModifiers());
 }
}
