import java.util.Date;
interface Namespace {
  class Source<X> { java.util.List<X> unsafe; }
}
interface Source<A> extends Namespace { A get(); }
abstract class Carrier<T> { abstract T get(); }
class Deferred<T> extends Carrier<T> {
  Source<T> source;
  T get() { return source.get(); }
}
class ExternalFactory {
  static <M extends Date> Source<M> make() {
    return new Source<M>() {
      public M get() { return null; }
    };
  }
}
