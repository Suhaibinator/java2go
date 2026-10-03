package transpiler

import "testing"

// Isolate the callable method boundary while the separate full probe retains
// its parameterized-constructor acceptance gate and all original observations.
func TestCI107GenericStaticReflectionMethodBoundaryJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `
import java.lang.reflect.Method;
public class Main {
 static class Base { int value; Base(int value){this.value=value;} }
 static final class Child extends Base { Child(int value){super(value);} }
 static int calls;
 public static synchronized <T> T identity(T value){calls++;return value;}
 public static <T extends Integer> int boxed(T value){calls++;return value+1;}
 public static <B extends Base,T extends B> int dependent(B first,T second){calls++; B widened=second; return first.value*100+widened.value;}
 public static <T> void touch(T value){calls++;}
 public static <T> int array(T... values){calls++;return values.length;}
 public static void main(String[] args) throws Exception {
  Method id=Main.class.getDeclaredMethod("identity",Object.class);
  Object value=new Child(3);
  System.out.println("identity:"+(id.invoke(null,value)==value)+":"+(id.invoke(null,new Object[]{null})==null)+":"+id.getReturnType().getName());
  System.out.println("boxed:"+Main.class.getDeclaredMethod("boxed",Integer.class).invoke(null,7));
  Method dependent=Main.class.getDeclaredMethod("dependent",Base.class,Base.class);
  System.out.println("dependent:"+dependent.invoke(null,new Base(2),new Child(5)));
  System.out.println("void:"+(Main.class.getDeclaredMethod("touch",Object.class).invoke(null,value)==null));
  System.out.println("array:"+Main.class.getDeclaredMethod("array",Object[].class).invoke(null,(Object)new String[]{"a","b"}));
  System.out.println("calls:"+calls);
  try { Main.class.getDeclaredMethod("boxed",Integer.class).invoke(null,new Object[]{null}); }
  catch(java.lang.reflect.InvocationTargetException expected) { System.out.println("cause:"+expected.getCause().getClass().getName()); }
  System.out.println("extra-calls:"+(calls-6));
 }
}`, "identity:true:true:java.lang.Object\nboxed:8\ndependent:205\nvoid:true\narray:2\ncalls:6\ncause:java.lang.NullPointerException\nextra-calls:1\n")
}
