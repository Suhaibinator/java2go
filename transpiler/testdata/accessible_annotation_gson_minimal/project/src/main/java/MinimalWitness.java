import access.fixture.Target;
import com.google.gson.FieldAttributes;
import com.google.gson.annotations.Expose;
import com.google.gson.annotations.SerializedName;
import com.google.gson.internal.ReflectionAccessFilterHelper;
import java.lang.annotation.Annotation;
import java.lang.reflect.AccessibleObject;
import java.lang.reflect.Field;
import java.lang.reflect.Method;
public final class MinimalWitness {
  public static class Model {
    @SerializedName(value="wire", alternate={"old"}) @Expose public String value;
  }
  public record AccessorOnly(String value) {
    @SerializedName("accessor-only") public String value() { return value; }
  }
  public static void main(String[] args) throws Exception {
    Target target = Target.fresh();
    Field field = Target.class.getDeclaredField("privateValue");
    AccessibleObject member = field;
    Method checker = AccessibleObject.class.getDeclaredMethod("canAccess", Object.class);
    System.out.println("before=" + member.canAccess(target) + ":" + checker.invoke(member, target)
        + ":gson=" + ReflectionAccessFilterHelper.canAccess(member, target));
    member.setAccessible(true);
    System.out.println("after=" + checker.invoke(member, target) + ":value=" + field.get(target));
    member.setAccessible(false);
    System.out.println("reset=" + checker.invoke(member, target));
    try { checker.invoke(member, (Object) null); }
    catch (Exception failure) {
      System.out.println("invalid=" + failure.getClass().getName() + ":" + failure.getMessage()
          + ":" + failure.getCause().getClass().getName() + ":" + failure.getCause().getMessage());
    }
    Field annotated = Model.class.getDeclaredField("value");
    for (Annotation annotation : new FieldAttributes(annotated).getAnnotations())
      System.out.println("annotation=" + annotation.annotationType().getName());
    SerializedName name = annotated.getAnnotation(SerializedName.class);
    String[] aliases = name.alternate(); aliases[0] = "mutated";
    System.out.println("metadata=" + name.value() + ":" + name.alternate()[0]);
    Method accessor = AccessorOnly.class.getDeclaredMethod("value");
    System.out.println("method-annotation=" + accessor.getAnnotation(SerializedName.class).value()
        + ":field-absent=" + (AccessorOnly.class.getDeclaredField("value").getAnnotation(SerializedName.class) == null));
  }
}
