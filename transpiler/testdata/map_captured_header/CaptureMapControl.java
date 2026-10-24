package staticabi;
import java.util.AbstractMap;
import java.util.Map;
import java.util.Set;
import java.util.Collections;
import java.util.function.Function;

// Header and body ownership must distinguish captured owner/method T from
// the local class's own T, even when Go emits T, T2 and T3 for those binders.
class CaptureOuter<T> {
  final T ownerValue;
  CaptureOuter(T value) { ownerValue=value; }
  <T> String inspect(T methodValue) {
    class Local<T> extends AbstractMap<String,T> implements Function<String,T> {
      T localValue;
      Local(T value) { localValue=value; }
      public T get(Object key) { return localValue; }
      public T put(String key,T value) { T old=localValue; localValue=value; return old; }
      public Set<Map.Entry<String,T>> entrySet() { return Collections.emptySet(); }
      public T apply(String key) { return localValue; }
      Object owner() { return CaptureOuter.this.ownerValue; }
      Object method() { return methodValue; }
    }
    Local<Integer> local=new Local<>(Integer.valueOf(9));
    Map<String,Integer> map=local;
    Map<String,Integer> target=new java.util.HashMap<>();
    Integer callback=target.computeIfAbsent("key",local);
    return (map.get("key").intValue()==9)+":"+(callback.intValue()==9)+":"+(local.owner()==ownerValue)+":"+(local.method()==methodValue);
  }
}
public class CaptureMapControl {
  public static void main(String[] args) {
    System.out.println(new CaptureOuter<String>("owner").<Long>inspect(Long.valueOf(4)));
  }
}
