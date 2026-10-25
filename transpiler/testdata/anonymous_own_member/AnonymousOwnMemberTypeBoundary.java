import java.util.Date;
class Box<X> {}
interface Source<A> { A get(); }
abstract class Carrier<T> { abstract T get(); }
class Deferred<T> extends Carrier<T> {
  Source<T> source;
  T get() { return source.get(); }
}
class ExternalFactory {
  static <M extends Date> Source<M> make() {
    return new Source<M>() {
      class Box<X> { java.util.List<X> unsafe; }
      Box<M> member;
      public M get() { return null; }
    };
  }
}
