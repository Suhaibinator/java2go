import java.util.Date;
interface Source<A> { A get(); }
abstract class Carrier<T> { abstract T get(); }
class Deferred<T> extends Carrier<T> {
  Source<T> source;
  T get() { return source.get(); }
}
class ExternalFactory {
  static <M extends Date> Source<M> make() {
    var input = new java.util.ArrayList<M>();
    return new Source<M>() {
      public M get() { input.toString(); return null; }
    };
  }
}
