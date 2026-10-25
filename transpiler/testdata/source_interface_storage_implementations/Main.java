package sourceinterfaceimplementations;

interface Source<A> { A construct(); }
final class ValueSource<A> implements Source<A> {
  A value;
  ValueSource(A value) { this.value=value; }
  public A construct() { return value; }
}
abstract class Carrier<A> { abstract A read(); abstract void write(A value); }
final class Deferred<A> extends Carrier<A> {
  private Source<A> source;
  Deferred(Source<A> source) { this.source=source; }
  A read() { return source.construct(); }
  void write(A value) { source=new ValueSource<A>(value); }
  Source<A> source() { return source; }
}
interface Factory { <M> Carrier<M> create(Source<M> source); }
final class Maker implements Factory {
  public <M> Carrier<M> create(Source<M> source) { return new Deferred<M>(source); }
}
@SuppressWarnings({"unchecked", "rawtypes"})
public class Main {
  public static void main(String[] args) {
    Source<String> original=new ValueSource<String>("ok");
    Factory factory=new Maker();
    Carrier<String> typed=factory.create(original);
    boolean same=((Deferred<String>)typed).source()==original;
    String first=typed.read();
    Carrier raw=typed;
    raw.write(Integer.valueOf(7));
    boolean stored=raw.read().equals(Integer.valueOf(7));
    boolean delayed=false;
    try { String invalid=typed.read(); } catch(ClassCastException expected) { delayed=true; }
    raw.write(null);
    boolean nil=typed.read()==null;
    Source<String> anonymous=new Source<String>() { public String construct(){return "anonymous";} };
    class Local implements Source<String> { public String construct(){return "local";} }
    boolean anon=factory.create(anonymous).read().equals("anonymous");
    boolean local=factory.create(new Local()).read().equals("local");
    Source<Source<String>> nested=new ValueSource<Source<String>>(original);
    boolean nestedIdentity=factory.create(nested).read()==original;
    System.out.println(first+"|"+same+"|"+stored+"|"+delayed+"|"+nil+"|"+anon+"|"+local+"|"+nestedIdentity);
  }
}
