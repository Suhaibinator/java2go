interface Source<A>{A get();}
abstract class Carrier<T>{abstract T get();}
class Deferred<T> extends Carrier<T>{Source<T> source;T get(){return source.get();}}
class ExternalFactory{
 static <M extends java.util.Date> Source<M> make(){
  return new Source<M>(){java.util.List<M> unsafe;public M get(){return null;}};
 }
}
