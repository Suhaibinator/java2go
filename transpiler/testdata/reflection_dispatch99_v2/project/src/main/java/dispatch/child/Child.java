package dispatch.child;
import dispatch.base.Base;
public class Child extends Base {
 protected String value() { return "child"; }
 public String local() { return "local-shadow"; }
 public String secret() { return "secret-child"; }
 public static String marker() { return "static-child"; }
 protected String value(int count) { return "overload-child:"+count; }
 protected String result() { return "result-child"; }
 protected String failure() { throw new IllegalStateException("child-failure"); }
}
