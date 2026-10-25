package records;
public class ReflectiveCanonicalProbe {
 public record Value(int number){}
 public static void main(String[] args)throws Exception {
  java.lang.reflect.Constructor<Value> ctor=Value.class.getDeclaredConstructor(int.class);
  try { Value made=(Value)ctor.newInstance(4); System.out.println("created="+made.number()); }
  catch(UnsupportedOperationException missing){System.out.println("created=unsupported");}
 }
}
