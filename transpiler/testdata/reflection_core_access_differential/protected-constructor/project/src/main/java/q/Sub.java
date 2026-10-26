package q;
import p.Base;import java.lang.reflect.Constructor;
public class Sub extends Base {
 public Sub(){super();}
 public static void main(String[] args)throws Exception {
  Constructor<?> c=Base.class.getDeclaredConstructor();boolean before=c.canAccess(null);String result;
  try {c.newInstance();result="constructed";}catch(IllegalAccessException expected){result="denied";}
  c.setAccessible(true);boolean after=c.canAccess(null);c.newInstance();
  System.out.println("ctor="+before+":"+result+":"+after+":constructed");
 }
}
