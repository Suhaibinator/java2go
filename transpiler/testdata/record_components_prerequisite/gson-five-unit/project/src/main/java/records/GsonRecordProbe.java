package records;
import com.google.gson.internal.reflect.ReflectionHelper;
import java.lang.reflect.Constructor;
import java.lang.reflect.Field;
import java.lang.reflect.Method;
public class GsonRecordProbe {
 public record Value(int count,String label){}
 public record Empty(){}
 public static class Plain{}
 public static void main(String[] args)throws Exception {
  System.out.println("record="+ReflectionHelper.isRecord(Value.class));
  System.out.println("ordinary="+ReflectionHelper.isRecord(Plain.class));
  String[] names=ReflectionHelper.getRecordComponentNames(Value.class);System.out.println("names="+names[0]+":"+names[1]);
  System.out.println("empty="+ReflectionHelper.getRecordComponentNames(Empty.class).length);
  Field field=Value.class.getDeclaredField("count");Method accessor=ReflectionHelper.getAccessor(Value.class,field);
  System.out.println("accessor="+accessor.invoke(new Value(7,"x")));
  Constructor<Value> constructor=ReflectionHelper.getCanonicalRecordConstructor(Value.class);
  System.out.println("count="+constructor.getParameterCount());Value built=constructor.newInstance(8,"new");
  System.out.println("built="+built.count()+":"+built.label());
  System.out.println("empty-count="+ReflectionHelper.getCanonicalRecordConstructor(Empty.class).getParameterCount());
 }
}
