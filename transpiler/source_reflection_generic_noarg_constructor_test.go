package transpiler

import "testing"

func TestSourceReflectionGenericNoargConstructorsJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `
public class Main {
 public static class Base {}
 public static class Unbounded {
  public static int calls;
  static {calls=10;}
  public int value;
  public <T> Unbounded(){value=++calls;}
 }
 public static class Dependent {
  public int value;
  public <B extends Base,T extends B> Dependent(){B first=null; T second=null; B widened=second; value=first==widened?7:0;}
 }
 public static class BoxedBound {
  public int value;
  public <T extends java.lang.Integer> BoxedBound(){T empty=null;value=empty==null?9:0;}
 }
 public static void main(String[] args) throws Exception {
  java.lang.reflect.Constructor<Unbounded> constructor=Unbounded.class.getConstructor();
  System.out.println("before:"+Unbounded.calls);
  Unbounded first=constructor.newInstance();
  Unbounded second=constructor.newInstance();
  System.out.println("unbounded:"+first.value+":"+second.value+":"+(first!=second));
  System.out.println("dependent:"+Dependent.class.getConstructor().newInstance().value);
  System.out.println("boxed:"+BoxedBound.class.getConstructor().newInstance().value);
 }
}`, "before:10\nunbounded:11:12:true\ndependent:7\nboxed:9\n")
}
