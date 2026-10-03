package transpiler

import "testing"

func TestSourceReflectionParameterizedConstructorRecordControlJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `
public class Main {
 public record Data(int value) {}
 public static class Holder { public int value; public Holder(){value=9;} }
 public static void main(String[] args) throws Exception {
  Data data=new Data(3);
  Holder holder=Holder.class.getConstructor().newInstance();
  System.out.println("record:"+data.value()+":"+holder.value);
 }
}`, "record:3:9\n")
}

func TestSourceReflectionParameterizedConstructorsJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `
import java.lang.reflect.Constructor;
public class Main {
 public static class Base { public int value; public Base(int value){this.value=value;} }
 public static class Child extends Base { public Child(int value){super(value);} }
 public static class Holder {
  public String result;
  public Holder(){result="empty";}
  public Holder(Integer value){result=value==null?"null":"boxed:"+value;}
  public Holder(long value){result="wide:"+value;}
  public Holder(Base value){result="source:"+value.value;}
  public Holder(String... values){result=values==null?"array:null":"array:"+values.length;}
 }
 public static class GenericCtor {
  public int value;
  public <B extends Base,T extends B> GenericCtor(B first,T second){B widened=second;value=first.value*100+widened.value;}
 }
 public static void main(String[] args) throws Exception {
  System.out.println(((Holder)Holder.class.getConstructor().newInstance()).result);
  Constructor<?> boxed=Holder.class.getConstructor(Integer.class);
  System.out.println(((Holder)boxed.newInstance(Integer.valueOf(7))).result);
  System.out.println(((Holder)boxed.newInstance(new Object[]{null})).result);
  Constructor<?> wide=Holder.class.getConstructor(long.class);
  System.out.println(((Holder)wide.newInstance(Integer.valueOf(8))).result);
  System.out.println(((Holder)Holder.class.getConstructor(Base.class).newInstance(new Child(9))).result);
  Constructor<?> array=Holder.class.getConstructor(String[].class);
  System.out.println(((Holder)array.newInstance((Object)new String[]{"a","b"})).result);
  System.out.println(((Holder)array.newInstance(new Object[]{null})).result);
  System.out.println("varargs:"+((array.getModifiers()&128)!=0));
  System.out.println("generic:"+((GenericCtor)GenericCtor.class.getConstructor(Base.class,Base.class).newInstance(new Base(2),new Child(5))).value);
  try {wide.newInstance(new Object[]{null});} catch(IllegalArgumentException expected){System.out.println("null-primitive");}
  try {wide.newInstance("bad");} catch(IllegalArgumentException expected){System.out.println("wrong-type");}
  try {boxed.newInstance();} catch(IllegalArgumentException expected){System.out.println("wrong-count");}
 }
}`, "empty\nboxed:7\nnull\nwide:8\nsource:9\narray:2\narray:null\nvarargs:true\ngeneric:205\nnull-primitive\nwrong-type\nwrong-count\n")
}

func TestSourceReflectionConstructorAccessCauseAndIdentityJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `
import java.lang.reflect.Constructor;
public class Main {
 public static class Integer { public int value; public Integer(int value){this.value=value;} }
 public static class Holder { public int value; public Holder(Integer value){this.value=value.value;} }
 public static class Throwing { public Throwing(java.lang.Integer value){int unboxed=value;} }
 public abstract static class Abstract { public Abstract(int value){} }
 public static void main(String[] args) throws Exception {
  Constructor<?> hidden=Secret.class.getDeclaredConstructor(int.class);
  try {hidden.newInstance(java.lang.Integer.valueOf(3));} catch(IllegalAccessException expected){System.out.println("denied");}
  hidden.setAccessible(true);
  System.out.println("accessible:"+((Secret)hidden.newInstance(java.lang.Integer.valueOf(4))).value);
  System.out.println("source:"+((Holder)Holder.class.getConstructor(Integer.class).newInstance(new Integer(5))).value);
  try {Holder.class.getConstructor(java.lang.Integer.class);} catch(NoSuchMethodException expected){System.out.println("namespace");}
  try {Throwing.class.getConstructor(java.lang.Integer.class).newInstance(new Object[]{null});}
  catch(java.lang.reflect.InvocationTargetException expected){System.out.println("cause:"+expected.getCause().getClass().getName());}
  try {Abstract.class.getConstructor(int.class).newInstance(java.lang.Integer.valueOf(1));}
  catch(InstantiationException expected){System.out.println("abstract");}
 }
}`, "denied\naccessible:4\nsource:5\nnamespace\ncause:java.lang.NullPointerException\nabstract\n", map[string]string{
		"Secret.java": `public class Secret { public int value; private Secret(int value){this.value=value;} }`,
	})
}
