import java.lang.annotation.*;
import java.lang.reflect.*;
import com.google.gson.FieldAttributes;
import com.google.gson.annotations.*;
public class AnnotationPrimitiveWitness {
 @Retention(RetentionPolicy.RUNTIME) @interface Version {double value() default 3.5;}
 @Retention(RetentionPolicy.RUNTIME) @interface Metadata {Class<?> kind();boolean enabled() default true;}
 @Version(1.25) @Metadata(kind=String.class,enabled=false) static class Model {
  @SerializedName(value="wire",alternate={"old","older"}) @Expose(serialize=false) @Version public String value;
  @SerializedName("defaults") @Expose public String defaults;
 }
 static class Child extends Model {}
 record AccessorOnly(String value) { @SerializedName("accessor-only") public String value(){return value;} }
 static class Constructed { @Version(2.5) Constructed(){} }
 static <A extends Annotation> A lookup(Field field,Class<A> type) {return field.getAnnotation(type);}
 public static void main(String[] args) throws Exception {
  Field field=Model.class.getDeclaredField("value");SerializedName name=lookup(field,SerializedName.class);
  Annotation erased=name;System.out.println("type="+erased.annotationType().getName());
  String[] aliases=name.alternate();aliases[0]="changed";System.out.println("aliases="+name.value()+":"+name.alternate()[0]+":"+name.alternate()[1]);
  Expose flags=field.getAnnotation(Expose.class);System.out.println("flags="+flags.serialize()+":"+flags.deserialize());
  Field defaults=Model.class.getDeclaredField("defaults");System.out.println("defaults="+defaults.getAnnotation(SerializedName.class).alternate().length+":"+defaults.getAnnotation(Expose.class).serialize()+":"+defaults.getAnnotation(Expose.class).deserialize());
  AccessibleObject base=field;System.out.println("base="+base.getAnnotation(SerializedName.class).value());
  Annotation[] all=field.getAnnotations();System.out.println("count="+all.length);all[0]=null;System.out.println("fresh="+(field.getAnnotations()[0]!=null));
  System.out.println("wrapper="+new FieldAttributes(field).getAnnotations().size()+":"+new FieldAttributes(field).getAnnotation(SerializedName.class).value());
  System.out.println("versions="+field.getAnnotation(Version.class).value()+":"+Model.class.getAnnotation(Version.class).value()+":"+(Child.class.getAnnotation(Version.class)==null));
  Metadata metadata=Model.class.getAnnotation(Metadata.class);System.out.println("metadata="+metadata.kind().getName()+":"+metadata.enabled());
  Method accessor=AccessorOnly.class.getDeclaredMethod("value");
  System.out.println("method="+accessor.getAnnotation(SerializedName.class).value()+":"+(AccessorOnly.class.getDeclaredField("value").getAnnotation(SerializedName.class)==null)+":"+accessor.getAnnotations().length);
  AccessorOnly record=new AccessorOnly("stored");Field recordField=AccessorOnly.class.getDeclaredField("value");recordField.setAccessible(true);
  System.out.println("record="+AccessorOnly.class.getSuperclass().getName()+":"+recordField.getModifiers()+":"+recordField.get(record)+":"+accessor.invoke(record));
  System.out.println("constructor="+Constructed.class.getDeclaredConstructor().getAnnotation(Version.class).value());
  try {accessor.getAnnotation(null);}catch(NullPointerException failure){System.out.println("null-method="+(failure.getMessage()==null));}
  try {field.getAnnotation(null);}catch(NullPointerException failure){System.out.println("null-field="+(failure.getMessage()==null));}
  try {Model.class.getAnnotation(null);}catch(NullPointerException failure){System.out.println("null-class="+(failure.getMessage()==null));}
 }
}
