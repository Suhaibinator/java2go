package observe;
import model.Value;
import model.Empty;
import model.Reserved;
import audit.Audit;
import java.lang.reflect.Constructor;
import java.lang.reflect.InvocationTargetException;
public class Main {
 public static void main(String[] args)throws Exception {
  Value a=new Value();System.out.println("a="+a.number()+":"+a.label());
  Value b=new Value(4);System.out.println("b="+b.number()+":"+b.label());
  Value c=new Value("z");System.out.println("c="+c.number()+":"+c.label());
  try{new Value(-1,"bad");}catch(IllegalArgumentException e){System.out.println("throw="+e.getMessage()+":"+Audit.current.number()+":"+(Audit.current.label()==null));}
  Constructor<Value> ctor=Value.class.getDeclaredConstructor(int.class,String.class);
  System.out.println("count="+ctor.getParameterCount());
  Value r=ctor.newInstance(5,"r");System.out.println("r="+r.number()+":"+r.label());
  try{ctor.newInstance(-2,"bad");}catch(InvocationTargetException e){System.out.println("wrapped="+e.getCause().getClass().getName()+":"+e.getCause().getMessage()+":"+Audit.current.number()+":"+(Audit.current.label()==null));}
  Empty e1=new Empty();Empty e2=new Empty();System.out.println("empty="+(e1!=e2)+":"+(e1.getClass()==e2.getClass())+":"+Empty.class.getDeclaredConstructor().getParameterCount());
  Reserved k=new Reserved(1,2,3);System.out.println("reserved="+k.type()+":"+k.map()+":"+k.func());
  System.out.println("calls="+Audit.calls);
 }
}
