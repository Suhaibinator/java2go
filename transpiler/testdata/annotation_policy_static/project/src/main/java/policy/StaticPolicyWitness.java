package policy;
import static java.lang.annotation.RetentionPolicy.*;
import static java.lang.annotation.ElementType.*;
@java.lang.annotation.Retention(value=RUNTIME) @java.lang.annotation.Target(value={FIELD,TYPE_USE}) @interface Mark {}
class Model {@Mark String value;}
@java.lang.annotation.Retention(value=RUNTIME) @java.lang.annotation.Target(value=TYPE) @interface ClassMark {}
@ClassMark class Marked {}
public class StaticPolicyWitness {
 public static void main(String[] args)throws Exception {
  System.out.println("field="+(Model.class.getDeclaredField("value").getAnnotation(Mark.class)!=null));
  System.out.println("class="+(Marked.class.getAnnotation(ClassMark.class)!=null));
 }
}
