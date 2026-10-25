import java.lang.annotation.*;
public class NestedAnnotationBoundary {
 public static int requested(){return Host.class.getAnnotation(Outer.class)==null?0:1;}
 public static int arrayRequested(){return ArrayHost.class.getAnnotation(OuterArray.class)==null?0:1;}
 public static int defaultRequested(){return DefaultHost.class.getAnnotation(OuterDefault.class)==null?0:1;}
 public static int allRequested(){return Host.class.getAnnotations().length;}
 public static void main(String[] args) {
  System.out.println("known="+Host.class.getAnnotation(Known.class).value());
  System.out.println("present="+Host.class.isAnnotationPresent(Outer.class));
  System.out.println("other="+(Host.class.getAnnotation(Other.class)==null));
  System.out.println("array="+ArrayHost.class.getAnnotation(Known.class).value()+":"+ArrayHost.class.isAnnotationPresent(OuterArray.class));
  System.out.println("default="+DefaultHost.class.getAnnotation(Known.class).value()+":"+DefaultHost.class.isAnnotationPresent(OuterDefault.class));
 }
}
@Retention(RetentionPolicy.RUNTIME) @interface Inner {int value();}
@Retention(RetentionPolicy.RUNTIME) @interface Outer {Inner value();}
@Retention(RetentionPolicy.RUNTIME) @interface OuterArray {Inner[] value();}
@Retention(RetentionPolicy.RUNTIME) @interface OuterDefault {Inner value() default @Inner(4);}
@Retention(RetentionPolicy.RUNTIME) @interface Known {String value();}
@Retention(RetentionPolicy.RUNTIME) @interface Other {}
@Outer(@Inner(1)) @Known("kept") class Host {}
@OuterArray({@Inner(2),@Inner(3)}) @Known("array") class ArrayHost {}
@OuterDefault @Known("default") class DefaultHost {}
