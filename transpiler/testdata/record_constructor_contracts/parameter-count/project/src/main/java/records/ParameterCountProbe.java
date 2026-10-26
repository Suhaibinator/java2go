package records;
public class ParameterCountProbe {
 static int initialized;
 static class Target {
  static { initialized++; }
  public Target(){}
  private Target(int number,String label){}
  public Target(String... labels){}
 }
 static class Constructor { int getParameterCount(){return 17;} }
 public static void main(String[] args)throws Exception {
  System.out.println("before="+initialized);
  java.lang.reflect.Constructor<?> zero=Target.class.getDeclaredConstructor();
  java.lang.reflect.Constructor<?> two=Target.class.getDeclaredConstructor(int.class,String.class);
  java.lang.reflect.Constructor<?> spread=Target.class.getDeclaredConstructor(String[].class);
  System.out.println("counts="+zero.getParameterCount()+":"+two.getParameterCount()+":"+spread.getParameterCount()+":"+initialized);
  System.out.println("source="+new Constructor().getParameterCount());
  java.lang.reflect.Constructor<?> absent=null;
  try { absent.getParameterCount(); } catch(NullPointerException expected){System.out.println("null=true");}
 }
}
