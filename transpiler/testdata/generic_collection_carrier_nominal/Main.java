package genericcollectioncarriernominal;
import java.util.ArrayList;
import java.util.Collection;
abstract class Carrier<A> { abstract A read(); abstract void write(A value); }
class Plain<A> extends Carrier<A> {
 A value;
 A read() { return value; }
 void write(A replacement) { value = replacement; }
}
class Many<E> extends Carrier<Collection<E>> {
 Collection<E> value;
 Collection<E> read() { return value; }
 void write(Collection<E> replacement) { value = replacement; }
}
class Token<A> {}
interface Factory { <M> Carrier<M> create(Token<M> token, M value); }
class Maker implements Factory {
 public <N> Carrier<N> create(Token<N> token, N value) {
  Plain<N> result = new Plain<N>(); result.write(value); return result;
 }
}
class OnlyIterable implements java.lang.Iterable<String> {
 int calls;
 public java.util.Iterator<String> iterator() { calls++; return new ArrayList<String>().iterator(); }
}
@SuppressWarnings({"rawtypes", "unchecked"})
public class Main {
 public static void main(String[] args) {
  Factory factory = new Maker();
  Carrier<String> text = factory.create(new Token<String>(), "ok");
  Many<String> many = new Many<String>();
  Collection<String> good = new ArrayList<String>(); good.add("clean");
  many.write(good);
  Carrier raw = many;
  boolean bridgeRejected = false;
  try { raw.write("wrong"); }
  catch (ClassCastException expected) { bridgeRejected = many.read() == good; }
  OnlyIterable only = new OnlyIterable();
  boolean nominalRejected = false;
  try { raw.write(only); }
  catch (ClassCastException expected) { nominalRejected = many.read() == good && only.calls == 0; }
  Collection polluted = new ArrayList<Object>(); polluted.add(Integer.valueOf(7));
  raw.write(polluted);
  boolean same = raw.read() == polluted && many.read() == polluted;
  boolean delayed = false;
  try { String wrong = many.read().iterator().next(); }
  catch (ClassCastException expected) { delayed = true; }
  raw.write(null);
  boolean absent = many.read() == null;
  System.out.println(text.read()+"|"+bridgeRejected+"|"+nominalRejected+"|"+same+"|"+delayed+"|"+absent);
 }
}
