package dispatch.base;
public class Base {
 protected String value() { return "base"; }
 String local() { return "local-base"; }
 private String secret() { return "secret-base"; }
 public final String fixed() { return "fixed-base"; }
 public static String marker() { return "static-base"; }
 protected String value(int count) { return "overload-base:"+count; }
 protected Object result() { return "result-base"; }
 protected String failure() { throw new IllegalStateException("base-failure"); }
}
