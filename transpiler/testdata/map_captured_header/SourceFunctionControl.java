package callbackabi;
import java.util.Map;
import java.util.HashMap;
import java.util.function.Function;
class Mapper<T> implements Function<String,T> {
  final T value;
  int calls;
  Mapper(T value) { this.value=value; }
  public T apply(String key) { calls++; return value; }
}
public class SourceFunctionControl {
  public static void main(String[] args) {
    Map<String,String> values=new HashMap<>();
    Mapper<String> mapper=new Mapper<>("made");
    String made=values.computeIfAbsent("key",mapper);
    String same=values.computeIfAbsent("key",mapper);
    System.out.println(made.equals("made")+":"+(made==same)+":"+(mapper.calls==1));
  }
}
