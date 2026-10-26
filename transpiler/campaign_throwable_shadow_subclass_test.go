package transpiler

import "testing"

func TestCampaignThrowableUnnamedShadowSubclassJVM(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `class NullPointerException extends RuntimeException {}
class Child extends NullPointerException {}
public class Main {
 public static void main(String[] args) {
  Child child=new Child();
  System.out.println(Child.class.getSuperclass().getName());
  System.out.println(Child.class.getSuperclass()==NullPointerException.class);
  System.out.println(NullPointerException.class.isAssignableFrom(Child.class));
  System.out.println(java.lang.NullPointerException.class.isAssignableFrom(Child.class));
  System.out.println(RuntimeException.class.isAssignableFrom(Child.class));
  System.out.println(child.getClass()==Child.class);
  try {throw child;} catch(NullPointerException caught) {System.out.println(caught==child);}
 }
}
`, "NullPointerException\ntrue\ntrue\nfalse\ntrue\ntrue\ntrue\n")
}
