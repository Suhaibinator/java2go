package dispatch.app;
import java.lang.reflect.Method;
import java.lang.reflect.InvocationTargetException;
import dispatch.base.Base;
import dispatch.child.Child;
import dispatch.child.UpgradedChild;
import dispatch.child.Implementor;
import dispatch.api.Root;
import dispatch.api.Leaf;
public class Main {
 public static void main(String[] args) throws Exception {
  Child child=new Child();
  Method value=Base.class.getDeclaredMethod("value"); value.setAccessible(true);
  System.out.println("protected:"+value.invoke(child));
  Method local=Base.class.getDeclaredMethod("local"); local.setAccessible(true);
  System.out.println("package-shadow:"+local.invoke(child));
  System.out.println("package-chain:"+local.invoke(new UpgradedChild()));
  Method secret=Base.class.getDeclaredMethod("secret"); secret.setAccessible(true);
  System.out.println("private:"+secret.invoke(child));
  System.out.println("final:"+Base.class.getMethod("fixed").invoke(child));
  System.out.println("class-static:"+Base.class.getMethod("marker").invoke(child));
  Method overload=Base.class.getDeclaredMethod("value",int.class); overload.setAccessible(true);
  System.out.println("overload:"+overload.invoke(child,Integer.valueOf(7)));
  Method result=Base.class.getDeclaredMethod("result"); result.setAccessible(true);
  System.out.println("bridge:"+result.invoke(child));
  Method failure=Base.class.getDeclaredMethod("failure"); failure.setAccessible(true);
  try { failure.invoke(child); } catch(InvocationTargetException expected) { System.out.println("target:"+expected.getCause().getMessage()); }
  try { value.invoke(child,Integer.valueOf(8)); } catch(IllegalArgumentException expected) { System.out.println("wrong-count"); }
  Method denied=Base.class.getDeclaredMethod("value");
  try { denied.invoke(child); } catch(IllegalAccessException expected) { System.out.println("denied"); }
  System.out.println("interface-direct:"+Root.class.getMethod("marker").invoke(null));
  try { Leaf.class.getMethod("marker"); System.out.println("leaf-static:present"); } catch(NoSuchMethodException expected) { System.out.println("leaf-static:absent"); }
  try { Implementor.class.getMethod("marker"); System.out.println("class-interface-static:present"); } catch(NoSuchMethodException expected) { System.out.println("class-interface-static:absent"); }
  System.out.println("default:"+Implementor.class.getMethod("label").invoke(new Implementor()));
  try { Child.class.getMethod("value"); System.out.println("protected-public:present"); } catch(NoSuchMethodException expected) { System.out.println("protected-public:absent"); }
 }
}
