package origin.app;
import java.lang.reflect.Method;
import java.lang.reflect.InvocationTargetException;
import origin.leaf.Child;
import origin.base.Base;
public class Main {
 public static void main(String[] args) throws Exception {
  Method bridge=Child.class.getDeclaredMethod("echo",String.class);
  Method body=Base.class.getDeclaredMethod("echo",Object.class);
  System.out.println("bridge:"+bridge.isBridge()+":"+bridge.isSynthetic()+":"+(bridge.getDeclaringClass()==Child.class)+":"+(bridge.getReturnType()==origin.real.Token.class));
  System.out.println("body:"+body.isBridge()+":"+body.isSynthetic()+":"+(body.getReturnType()==origin.real.Token.class));
  Child child=new Child();
  origin.real.Token token=(origin.real.Token)bridge.invoke(child,"ok");
  System.out.println("invoke:"+token.value+":"+child.calls);
  try {bridge.invoke(child,Integer.valueOf(7));} catch(IllegalArgumentException expected){System.out.println("rejected:"+child.calls);}
  origin.real.Token erased=(origin.real.Token)body.invoke(child,"body");
  System.out.println("erased:"+erased.value+":"+child.calls);
 }
}
