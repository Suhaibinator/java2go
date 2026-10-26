import java.util.Date;
class Box<X> {}
interface Source<A> {
  A get();
  class Box<X> { java.util.List<X> unsafe; }
}
abstract class Carrier<T> { abstract T get(); }
class Deferred<T> extends Carrier<T> {
  Source<T> source;
  T get() { return source.get(); }
}
class ExternalFactory {
  static <M extends Date> Source<M> make() {
    return new Source<M>() {
      Box<M> inherited;
      public M get() { return null; }
    };
  }
}
