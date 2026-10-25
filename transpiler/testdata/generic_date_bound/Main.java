package genericdate;
import java.util.Date;
abstract class Base<A> { abstract A read(); abstract void write(A value); }
class Plain<A> extends Base<A> {
  A stored;
  A read() { return stored; }
  void write(A value) { stored = value; }
}
class DateChild<D extends Date> extends Base<D> {
  D stored;
  D read() { return stored; }
  void write(D value) { stored = value; }
}
class Token<A> {}
interface Factory { <M> Base<M> create(Token<M> token, M value); }
class Maker implements Factory {
  public <X> Base<X> create(Token<X> token, X value) {
    Plain<X> box = new Plain<X>(); box.write(value); return box;
  }
}
public class Main {
  public static void main(String[] args) {
    Factory factory = new Maker();
    Base<String> text = factory.create(new Token<String>(), "s");
    Base<Integer> number = factory.create(new Token<Integer>(), Integer.valueOf(17));
    Base<String> empty = factory.create(new Token<String>(), null);
    DateChild<java.sql.Timestamp> dated = new DateChild<java.sql.Timestamp>();
    java.sql.Timestamp stamp = new java.sql.Timestamp(1234L);
    dated.write(stamp);
    Base<java.sql.Timestamp> view = dated;
    Date nativeView = view.read();
    boolean same = nativeView == stamp;
    nativeView.setTime(5678L);
    System.out.println(text.read()+"|"+number.read()+"|"+(empty.read()==null)+"|"+same+"|"+view.read().getTime()+"|"+view.read().getNanos()+"|"+(nativeView instanceof java.sql.Timestamp));
  }
}
