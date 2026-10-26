package records;
import java.lang.reflect.Method;
import java.lang.reflect.RecordComponent;
public class DynamicProbe {
 public record Value(int count,String label){public String label(){return label;}}
 public record Empty(){}
 public static class Plain{}
 static void lookup(String tag,Class<?> owner,String name){
  try{Method method=owner.getMethod(name);System.out.println(tag+"="+method.getDeclaringClass().getName()+"#"+method.getName());}
  catch(Exception e){System.out.println(tag+"="+e.getClass().getName());}
 }
 public static void main(String[] args)throws Exception {
  lookup("lookup-flag",Class.class,"isRecord");
  lookup("lookup-components",Class.class,"getRecordComponents");
  lookup("lookup-name",RecordComponent.class,"getName");
  lookup("lookup-type",RecordComponent.class,"getType");
  try{System.out.println("load="+Class.forName("java.lang.reflect.RecordComponent").getName());}
  catch(Exception e){System.out.println("load="+e.getClass().getName());}
  try{Method flag=Class.class.getMethod("isRecord");System.out.println("flags="+flag.invoke(Value.class)+":"+flag.invoke(Empty.class)+":"+flag.invoke(Plain.class)+":"+flag.invoke(Record.class)+":"+flag.invoke(int.class)+":"+flag.invoke(String[].class));}
  catch(Exception e){System.out.println("flags="+e.getClass().getName());}
  try{
   Method components=Class.class.getMethod("getRecordComponents");
   Method name=RecordComponent.class.getMethod("getName");Method type=RecordComponent.class.getMethod("getType");
   Object[] first=(Object[])components.invoke(Value.class);
   for(int i=0;i<first.length;i++){System.out.println("component"+i+"="+name.invoke(first[i])+":"+((Class<?>)type.invoke(first[i])).getName());}
   first[0]=null;Object[] next=(Object[])components.invoke(Value.class);System.out.println("fresh="+(next.length==2 && next[0]!=null));
   Object[] empty=(Object[])components.invoke(Empty.class);System.out.println("empty="+empty.length);
   Object[] emptyAgain=(Object[])components.invoke(Empty.class);System.out.println("empty-fresh="+(empty!=emptyAgain));
   System.out.println("ordinary-null="+(components.invoke(Plain.class)==null));
  }catch(Exception e){System.out.println("components="+e.getClass().getName());}
  try{System.out.println("implicit="+Value.class.getMethod("count").invoke(new Value(7,"x")));}
  catch(Exception e){System.out.println("implicit="+e.getClass().getName());}
  try{System.out.println("explicit="+Value.class.getMethod("label").invoke(new Value(7,"x")));}
  catch(Exception e){System.out.println("explicit="+e.getClass().getName());}
  System.out.println("canonical="+Value.class.getDeclaredConstructor(int.class,String.class).getParameterCount());
 }
}
