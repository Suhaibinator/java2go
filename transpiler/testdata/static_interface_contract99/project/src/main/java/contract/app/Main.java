package contract.app;
interface StaticOnly { static String marker() { return secret(); } static <T> T keep(T value) { return value; } private static String secret() { return "static-only"; } }
interface DefaultOnly { default String label() { return helper(); } private String helper() { return "default-only"; } static String marker() { return "default-static"; } }
class DefaultImpl implements DefaultOnly {}
interface Mixed { String apply(String value); default String decorate(String value) { return prefix()+apply(value); } private String prefix() { return "mixed:"; } static String marker() { return "mixed-static"; } }
interface Inherited extends Left {}
class InheritedImpl implements Inherited {}
interface Left { default String label() { return "left"; } }
interface Right { default String label() { return "right"; } }
class Conflict implements Left, Right { public String label() { return "resolved"; } }
interface Generic<T> { T apply(T value); default T identity(T value) { return apply(value); } static String marker() { return "generic-static"; } private static String helper() { return "unused"; } }
interface Wide { Object result(); }
interface Narrow extends Wide { String result(); }
class NarrowImpl implements Narrow { public String result() { return "narrow"; } }
public class Main {
 public static void main(String[] args) {
  System.out.println(StaticOnly.keep(StaticOnly.marker()));
  DefaultOnly defaults = new DefaultImpl();
  System.out.println(defaults.label()+":"+DefaultOnly.marker());
  Mixed mixed = value -> value;
  System.out.println(mixed.decorate("value")+":"+Mixed.marker());
  System.out.println(new Conflict().label()+":"+new InheritedImpl().label());
  Generic<String> generic = value -> value;
  System.out.println(generic.identity("generic")+":"+Generic.marker());
  Wide wide = new NarrowImpl();
  System.out.println(wide.result());
 }
}
